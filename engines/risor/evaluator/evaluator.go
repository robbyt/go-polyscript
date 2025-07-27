package evaluator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	risorLib "github.com/risor-io/risor"
	risorCompiler "github.com/risor-io/risor/compiler"
	"github.com/robbyt/go-polyscript/engines/risor/internal"
	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script"
)

// Evaluator is an abstraction layer for evaluating bytecode on the Risor engine
type Evaluator struct {
	// ctxKey is the variable name used to access input data inside the engine (ctx)
	ctxKey string

	// execUnit contains the compiled script and data provider
	execUnit *script.ExecutableUnit

	logHandler slog.Handler
	logger     *slog.Logger
}

// New creates a new Evaluator object
func New(
	handler slog.Handler,
	execUnit *script.ExecutableUnit,
) *Evaluator {
	handler, logger := helpers.SetupLogger(handler, "risor", "Evaluator")

	return &Evaluator{
		ctxKey:     constants.Ctx,
		execUnit:   execUnit,
		logHandler: handler,
		logger:     logger,
	}
}

func (be *Evaluator) String() string {
	return "risor.Evaluator"
}

// loadInputData retrieves input data using the data provider in the executable unit.
// Returns a map that will be used as input for the Risor engine.
func (be *Evaluator) loadInputData(ctx context.Context) (map[string]any, error) {
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
func (be *Evaluator) exec(
	ctx context.Context,
	bytecode *risorCompiler.Code,
	options ...risorLib.Option,
) (*execResult, error) {
	startTime := time.Now()
	result, err := risorLib.EvalCode(ctx, bytecode, options...)
	execTime := time.Since(startTime)

	if err != nil {
		return nil, fmt.Errorf("risor execution error: %w", err)
	}
	return newEvalResult(be.logHandler, result, execTime, ""), nil
}

// Eval evaluates the loaded bytecode and uses the provided EvalData to pass data in to the Risor engine execution
func (be *Evaluator) Eval(ctx context.Context) (platform.EvaluatorResponse, error) {
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

	// 3. Convert to Risor engine format
	runtimeData := internal.ConvertToRisorOptions(be.ctxKey, rawInputData)

	// 4. Execute the program
	result, err := be.exec(ctx, risorByteCode, runtimeData...)
	if err != nil {
		return nil, fmt.Errorf("exec error: %w", err)
	}
	logger.DebugContext(ctx, "exec complete", "result", result)

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

	return result, nil
}

// AddDataToContext implements the data.Setter interface which stores and prepares runtime data
// which can be eventually passed to the Eval method.
func (be *Evaluator) AddDataToContext(
	ctx context.Context,
	d ...map[string]any,
) (context.Context, error) {
	logger := be.logger.WithGroup("AddDataToContext")

	// Use the shared helper function for context preparation
	if be.execUnit == nil || be.execUnit.GetDataProvider() == nil {
		return ctx, fmt.Errorf("no data provider available")
	}

	return data.AddDataToContextHelper(
		ctx,
		logger,
		be.execUnit.GetDataProvider(),
		d...,
	)
}
