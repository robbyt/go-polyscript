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
func NewBytecodeEvaluator(
	handler slog.Handler,
	execUnit *script.ExecutableUnit,
) *BytecodeEvaluator {
	handler, logger := helpers.SetupLogger(handler, "risor", "BytecodeEvaluator")

	return &BytecodeEvaluator{
		ctxKey:     constants.Ctx,
		execUnit:   execUnit,
		logHandler: handler,
		logger:     logger,
	}
}

func (be *BytecodeEvaluator) String() string {
	return "risor.BytecodeEvaluator"
}

// loadInputData retrieves input data using the data provider in the executable unit.
// Returns a map that will be used as input for the Risor VM.
func (be *BytecodeEvaluator) loadInputData(ctx context.Context) (map[string]any, error) {
	logger := be.logger.WithGroup("loadInputData")

	// If no executable unit or data provider, return empty map
	if be.execUnit == nil || be.execUnit.GetDataProvider() == nil {
		logger.WarnContext(ctx, "no data provider available, using empty data")
		return make(map[string]any), nil
	}

	// Get input data from provider
	inputData, err := be.execUnit.GetDataProvider().GetData(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get input data from provider", "error", err)
		return nil, err
	}

	if len(inputData) == 0 {
		logger.WarnContext(ctx, "empty input data returned from provider")
	}
	logger.DebugContext(ctx, "input data loaded from provider", "inputData", inputData)
	return inputData, nil
}

// exec pulls the latest bytecode, and runs it with some input from options
func (be *BytecodeEvaluator) exec(
	ctx context.Context,
	bytecode *risorCompiler.Code,
	options ...risorLib.Option,
) (*execResult, error) {
	logger := be.logger.WithGroup("exec")
	startTime := time.Now()
	result, err := risorLib.EvalCode(ctx, bytecode, options...)
	execTime := time.Since(startTime)

	if err != nil {
		return nil, fmt.Errorf("risor execution error: %w", err)
	}

	logger.InfoContext(ctx, "execution complete", "result", result)
	return newEvalResult(be.logHandler, result, execTime, ""), nil
}

// Eval evaluates the loaded bytecode and uses the provided EvalData to pass data in to the Risor VM execution
func (be *BytecodeEvaluator) Eval(ctx context.Context) (engine.EvaluatorResponse, error) {
	logger := be.logger.WithGroup("Eval")
	if be.execUnit == nil {
		return nil, fmt.Errorf("executable unit is nil")
	}

	if be.execUnit.GetContent() == nil {
		return nil, fmt.Errorf("content is nil")
	}

	// Get the bytecode from the executable unit
	bytecode := be.execUnit.GetContent().GetByteCode()
	if bytecode == nil {
		return nil, fmt.Errorf("bytecode is nil")
	}

	// Get execution ID
	exeID := be.execUnit.GetID()
	if exeID == "" {
		return nil, fmt.Errorf("exeID is empty")
	}
	logger = logger.With("exeID", exeID)

	// 1. Type assert the bytecode into *risorCompiler.Code
	risorByteCode, ok := bytecode.(*risorCompiler.Code)
	if !ok {
		return nil, fmt.Errorf(
			"unable to type assert bytecode into *risorCompiler.Code for ID: %s",
			exeID,
		)
	}

	// 2. Get the raw input data
	rawInputData, err := be.loadInputData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get input data: %w", err)
	}

	// 3. Convert to Risor VM format
	runtimeData := convertToRisorOptions(be.ctxKey, rawInputData)

	// 4. Execute the program
	result, err := be.exec(ctx, risorByteCode, runtimeData...)
	if err != nil {
		return nil, fmt.Errorf("error returned from script: %w", err)
	}
	logger.Debug("script execution complete", "result", result)

	// 5. Collect results
	result.scriptExeID = exeID

	if result.Object == nil {
		logger.Warn("result object is nil")
		return result, nil
	}

	switch result.Object.Type() {
	case "error":
		return result, fmt.Errorf("error returned from script: %s", result.Inspect())
	case "function":
		return result, fmt.Errorf("function object returned from script: %s", result.Inspect())
	}

	logger.DebugContext(ctx, "execution complete")
	return result, nil
}

// PrepareContext implements the EvalDataPreparer interface for Risor scripts.
// It enriches the provided context with data for script evaluation, using the
// ExecutableUnit's DataProvider to store the data.
func (be *BytecodeEvaluator) PrepareContext(
	ctx context.Context,
	d ...any,
) (context.Context, error) {
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
