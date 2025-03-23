package risor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	risorLib "github.com/risor-io/risor"
	risorCompiler "github.com/risor-io/risor/compiler"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/internal/helpers"
)

// BytecodeEvaluator is an abstraction layer for evaluating bytecode on the Risor VM
type BytecodeEvaluator struct {
	// ctxKey is the variable name used to access input data inside the vm (ctx)
	ctxKey string

	// execUnit contains the compiled script and data provider
	execUnit *script.ExecutableUnit

	logHandler slog.Handler
	logger     *slog.Logger
}

// NewBytecodeEvaluator creates a new BytecodeEvaluator object
func NewBytecodeEvaluator(handler slog.Handler, execUnit *script.ExecutableUnit) *BytecodeEvaluator {
	handler, logger := helpers.SetupLogger(handler, "risor", "BytecodeEvaluator")

	return &BytecodeEvaluator{
		ctxKey:     constants.Ctx,
		execUnit:   execUnit,
		logHandler: handler,
		logger:     logger,
	}
}

func (c *BytecodeEvaluator) String() string {
	return "risor.BytecodeEvaluator"
}

// exec pulls the latest bytecode, and runs it with some input from options
func (be *BytecodeEvaluator) exec(ctx context.Context, bytecode *risorCompiler.Code, options ...risorLib.Option) (*execResult, error) {
	startTime := time.Now()
	result, err := risorLib.EvalCode(ctx, bytecode, options...)
	execTime := time.Since(startTime)

	if err != nil {
		return nil, err
	}

	return newEvalResult(be.logHandler, result, execTime, ""), nil
}

// convertInputData creates a slice of risorLib.Option objects from the input data.
// The input data will be wrapped in a single "ctx" object passed to the VM.
//
// For example, if the inputData is {"foo": "bar", "baz": 123}, the output will be:
//
//	[]risorLib.Option{
//	  risorLib.WithGlobal("ctx", map[string]any{
//	    "foo": "bar",
//	    "baz": 123,
//	  }),
//	}
func (be *BytecodeEvaluator) convertInputData(ctx context.Context) ([]risorLib.Option, error) {
	// setup input data, which will be sent from the input request to the eval VM
	logger := be.logger.WithGroup("convertInputData")

	if be.execUnit == nil || be.execUnit.GetDataProvider() == nil {
		logger.WarnContext(ctx, "no data provider available, using empty data")
		return []risorLib.Option{
			risorLib.WithGlobal(be.ctxKey, map[string]any{}),
		}, nil
	}

	// Use the data provider to get the input data
	inputData, err := be.execUnit.GetDataProvider().GetData(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get input data from provider", "error", err)
		return []risorLib.Option{}, err
	}

	if len(inputData) == 0 {
		logger.WarnContext(ctx, "empty input data returned from provider")
	} else {
		logger.DebugContext(ctx, "input data loaded from provider", "inputData", inputData)
	}

	return []risorLib.Option{
		risorLib.WithGlobal(be.ctxKey, inputData),
	}, nil
}

// Eval evaluates the loaded bytecode and uses the provided EvalData to pass data in to the Risor VM execution
func (be *BytecodeEvaluator) Eval(ctx context.Context) (engine.EvaluatorResponse, error) {
	logger := be.logger.WithGroup("Eval")
	if be.execUnit == nil {
		return nil, fmt.Errorf("executable unit is nil")
	}

	// Get the bytecode from the executable unit
	bytecode := be.execUnit.GetContent().GetByteCode()
	if bytecode == nil {
		return nil, fmt.Errorf("bytecode is nil")
	}

	exeID := be.execUnit.GetID()
	if exeID == "" {
		return nil, fmt.Errorf("exeID is empty")
	}
	logger = logger.With("exeID", exeID)

	// Get input data options using the data provider
	inputDataOptions, err := be.convertInputData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get input data: %w", err)
	}

	// assert the bytecode into *risorCompiler.Code that can be run by *this* BytecodeEvaluator
	risorByteCode, ok := bytecode.(*risorCompiler.Code)
	if !ok {
		return nil, fmt.Errorf("unable to type assert bytecode into *risorCompiler.Code for ID: %s", exeID)
	}

	// Evaluate the bytecode, passing the options
	result, err := be.exec(ctx, risorByteCode, inputDataOptions...)
	if err != nil {
		return nil, fmt.Errorf("error returned from script: %w", err)
	}
	logger.Debug("script execution complete", "result", result)

	// Set the script version on the result
	result.scriptExeID = exeID

	if result.Object == nil {
		logger.Warn("result object is nil")
		return result, nil
	}

	switch result.Object.Type() {
	case "error":
		return result, fmt.Errorf("error returned from script: %s", result.Object.Inspect())
	case "function":
		return result, fmt.Errorf("function object returned from script: %s", result.Object.Inspect())
	}

	return result, nil
}

// PrepareContext implements the EvalDataPreparer interface for Risor scripts.
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
