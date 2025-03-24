package starlark

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/internal/helpers"

	starlarkLib "go.starlark.net/starlark"
)

// BytecodeEvaluator is an abstraction layer for evaluating code on the Starlark VM
type BytecodeEvaluator struct {
	// universe is the global variable map for the Starlark VM
	universe starlarkLib.StringDict

	// execUnit contains the compiled script and data provider
	execUnit *script.ExecutableUnit

	logHandler slog.Handler
	logger     *slog.Logger
}

// NewBytecodeEvaluator creates a new BytecodeEvaluator object
func NewBytecodeEvaluator(handler slog.Handler, execUnit *script.ExecutableUnit) *BytecodeEvaluator {
	handler, logger := helpers.SetupLogger(handler, "starlark", "BytecodeEvaluator")

	// Get universe with standard modules
	universe := standardModules()

	// Add eval-time contextual globals
	universe[constants.Ctx] = starlarkLib.None
	universe[string(constants.EvalData)] = starlarkLib.None

	return &BytecodeEvaluator{
		universe:   universe,
		execUnit:   execUnit,
		logHandler: handler,
		logger:     logger,
	}
}

func (be *BytecodeEvaluator) String() string {
	return "starlark.BytecodeEvaluator"
}

// prepareGlobals merges the universe and input globals into a single Starlark dictionary
func (be *BytecodeEvaluator) prepareGlobals(ctx context.Context, inputGlobals starlarkLib.StringDict) starlarkLib.StringDict {
	// Pre-allocate with exact capacity needed
	mergedGlobals := make(starlarkLib.StringDict, len(be.universe)+len(inputGlobals))

	// Copy the pre-populated universe first
	maps.Copy(mergedGlobals, be.universe)

	// Then add execution-specific globals, which may override universe values
	maps.Copy(mergedGlobals, inputGlobals)

	return mergedGlobals
}

// exec executes the bytecode with the provided globals
func (be *BytecodeEvaluator) exec(ctx context.Context, prog *starlarkLib.Program, globals starlarkLib.StringDict) (*execResult, error) {
	logger := be.logger.WithGroup("exec")
	startTime := time.Now()

	// Create thread with cancellation support
	thread := &starlarkLib.Thread{
		Name: "eval",
		Print: func(thread *starlarkLib.Thread, msg string) {
			logger.InfoContext(ctx, msg, "starlark-thread", thread.Name)
		},
	}

	// Set up cancellation from context
	go func() {
		<-ctx.Done()
		thread.Cancel(ctx.Err().Error())
	}()

	// Execute the program
	finalGlobals, err := prog.Init(thread, globals)
	execTime := time.Since(startTime)

	if err != nil {
		return nil, fmt.Errorf("starlark execution error: %w", err)
	}

	// Get the main value from globals
	// The "_" key contains the last value evaluated in the Starlark script
	mainVal := finalGlobals["_"]
	if mainVal == nil {
		mainVal = starlarkLib.None
	}
	logger.InfoContext(ctx, "execution complete", "mainVal", mainVal)

	// Check if the value is None and try to find a valid result variable
	if mainVal == starlarkLib.None {
		// Look for a variable named "result" which is a common pattern
		if resultVal, ok := finalGlobals["result"]; ok {
			logger.InfoContext(ctx, "found explicit result variable", "result", resultVal)
			mainVal = resultVal
		}
	}

	return newEvalResult(be.logHandler, mainVal, execTime, ""), nil
}

// Eval evaluates the loaded bytecode and passes the provided data into the Starlark VM
func (be *BytecodeEvaluator) Eval(ctx context.Context) (engine.EvaluatorResponse, error) {
	logger := be.logger.WithGroup("Eval")
	if be.execUnit == nil {
		return nil, fmt.Errorf("executable unit is nil")
	}

	// Get bytecode from executable unit
	bytecode := be.execUnit.GetContent().GetByteCode()
	if bytecode == nil {
		return nil, fmt.Errorf("bytecode is nil")
	}

	// Type assert to Starlark program
	prog, ok := bytecode.(*starlarkLib.Program)
	if !ok {
		return nil, fmt.Errorf("invalid bytecode type: expected *starlark.Program, got %T", bytecode)
	}

	exeID := be.execUnit.GetID()
	if exeID == "" {
		return nil, fmt.Errorf("execution ID is empty")
	}
	logger = logger.With("exeID", exeID)

	// Get input data from the provider in the executable unit
	var inputData map[string]any
	var err error

	if be.execUnit.GetDataProvider() != nil {
		inputData, err = be.execUnit.GetDataProvider().GetData(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get input data from provider: %w", err)
		}
	} else {
		// If no provider is specified, use an empty map
		inputData = make(map[string]any)
	}

	// Convert input data to Starlark values
	globals, err := convertInputData(inputData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input data: %w", err)
	}

	// Prepare globals by merging with universe
	mergedGlobals := be.prepareGlobals(ctx, globals)

	// Execute the program
	result, err := be.exec(ctx, prog, mergedGlobals)
	if err != nil {
		return nil, fmt.Errorf("execution error: %w", err)
	}
	logger.Debug("script execution complete", "result", result)

	// Set the execution ID
	result.scriptExeID = exeID

	// Handle specific return types
	if result.Value == nil {
		logger.Warn("result value is nil")
		return result, nil
	}

	// Handle callable results (functions)
	if callable, ok := result.Value.(starlarkLib.Callable); ok {
		thread := &starlarkLib.Thread{Name: "func"}
		val, err := starlarkLib.Call(thread, callable, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("error calling function: %w", err)
		}
		// "Freeze" the value to prevent any further modifications
		val.Freeze()
		return newEvalResult(be.logHandler, val, result.execTime, exeID), nil
	}

	return result, nil
}

// PrepareContext implements the EvalDataPreparer interface for Starlark scripts.
// It enriches the provided context with data for script evaluation, using the
// ExecutableUnit's DataProvider to store the data.
func (be *BytecodeEvaluator) PrepareContext(ctx context.Context, d ...any) (context.Context, error) {
	logger := be.logger.WithGroup("PrepareContext")

	// Use the shared helper function for context preparation
	if be.execUnit == nil || be.execUnit.GetDataProvider() == nil {
		return ctx, fmt.Errorf("no data provider available")
	}

	return data.PrepareContextHelper(
		ctx,
		logger,
		be.execUnit.GetDataProvider(),
		d...,
	)
}
