package starlark

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"time"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/script"

	starlarkLib "go.starlark.net/starlark"
)

// BytecodeEvaluator is an abstraction layer for evaluating code on the Starlark VM
type BytecodeEvaluator struct {
	// evalDataKey is the variable used to load input data from the context object (eval_data)
	evalDataKey string

	// universe is the global variable map for the Starlark VM
	universe starlarkLib.StringDict

	logHandler slog.Handler
	logger     *slog.Logger
}

// NewBytecodeEvaluator creates a new BytecodeEvaluator object
func NewBytecodeEvaluator(handler slog.Handler) *BytecodeEvaluator {
	if handler == nil {
		defaultHandler := slog.NewTextHandler(os.Stdout, nil)
		handler = defaultHandler.WithGroup("starlark")
		slog.New(handler).Warn("Handler is nil, using the default logger configuration.")
	}

	// Get universe with standard modules
	universe := standardModules()

	// Add eval-time contextual globals
	universe[constants.Ctx] = starlarkLib.None
	universe[constants.EvalData] = starlarkLib.None

	return &BytecodeEvaluator{
		evalDataKey: constants.EvalData,
		universe:    universe,
		logHandler:  handler,
		logger:      slog.New(handler.WithGroup("BytecodeEvaluator")),
	}
}

func (r *BytecodeEvaluator) getLogger() *slog.Logger {
	return r.logger
}

func (c *BytecodeEvaluator) String() string {
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
	logger := be.getLogger()
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
	mainVal := finalGlobals["_"]
	if mainVal == nil {
		mainVal = starlarkLib.None
	}
	logger.InfoContext(ctx, "execution complete", "mainVal", mainVal)

	return newEvalResult(be.logHandler, mainVal, execTime, ""), nil
}

// Eval evaluates the loaded bytecode and passes the provided data into the Starlark VM
func (be *BytecodeEvaluator) Eval(ctx context.Context, exe *script.ExecutableUnit) (engine.EvaluatorResponse, error) {
	logger := be.getLogger()
	if exe == nil {
		return nil, fmt.Errorf("executable unit is nil")
	}

	// Get bytecode from executable unit
	bytecode := exe.GetContent().GetByteCode()
	if bytecode == nil {
		return nil, fmt.Errorf("bytecode is nil")
	}

	// Type assert to Starlark program
	prog, ok := bytecode.(*starlarkLib.Program)
	if !ok {
		return nil, fmt.Errorf("invalid bytecode type: expected *starlark.Program, got %T", bytecode)
	}

	exeID := exe.GetID()
	if exeID == "" {
		return nil, fmt.Errorf("execution ID is empty")
	}
	logger = logger.With("exeID", exeID)

	inputData := make(map[string]any)
	if ctxData := ctx.Value(be.evalDataKey); ctxData != nil {
		inputData, ok = ctxData.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid input data type: expected map[string]any, got %T", ctxData)
		}
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
