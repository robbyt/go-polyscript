package risor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	risorLib "github.com/risor-io/risor"
	risorCompiler "github.com/risor-io/risor/compiler"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/script"
)

// BytecodeEvaluator is an abstraction layer for evaluating bytecode on the Risor VM
type BytecodeEvaluator struct {
	// ctxKey is the variable name used to access input data inside the vm (ctx)
	ctxKey string

	// evalDataKey is the variable used to load input data from the context object (eval_data)
	evalDataKey string

	logHandler slog.Handler
	logger     *slog.Logger
}

// NewBytecodeEvaluator creates a new BytecodeEvaluator object
func NewBytecodeEvaluator(handler slog.Handler) *BytecodeEvaluator {
	if handler == nil {
		defaultHandler := slog.NewTextHandler(os.Stdout, nil)
		handler = defaultHandler.WithGroup("risor")
		slog.New(handler).Warn("Handler is nil, using the default logger configuration.")
	}

	return &BytecodeEvaluator{
		ctxKey:      constants.Ctx,
		evalDataKey: constants.EvalData,
		logHandler:  handler,
		logger:      slog.New(handler.WithGroup("BytecodeEvaluator")),
	}
}

func (r *BytecodeEvaluator) getLogger() *slog.Logger {
	return r.logger
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
func (be *BytecodeEvaluator) convertInputData(ctx context.Context) []risorLib.Option {
	// setup input data, which will be sent from the input request to the eval VM
	logger := be.getLogger()

	ctxData := ctx.Value(be.evalDataKey)
	if ctxData == nil {
		logger.ErrorContext(ctx, "input data not found in context")
		return []risorLib.Option{}
	}

	inputData, ok := ctxData.(map[string]any)
	if !ok {
		logger.ErrorContext(ctx, "invalid input data type: expected map[string]any", "ctxData", ctxData)
		return []risorLib.Option{}
	}

	logger.DebugContext(ctx, "input data loaded", "inputData", inputData, "contextStorageKey", be.evalDataKey)
	return []risorLib.Option{
		risorLib.WithGlobal(be.ctxKey, inputData),
	}
}

// Eval evaluates the loaded bytecode and uses the provided EvalData to pass data in to the Risor VM execution
func (be *BytecodeEvaluator) Eval(ctx context.Context, exe *script.ExecutableUnit) (engine.EvaluatorResponse, error) {
	logger := be.getLogger()
	if exe == nil {
		return nil, fmt.Errorf("version is nil")
	}

	// Get the bytecode from the executable unit
	bytecode := exe.GetContent().GetByteCode()
	if bytecode == nil {
		return nil, fmt.Errorf("bytecode is nil")
	}

	exeID := exe.GetID()
	if exeID == "" {
		return nil, fmt.Errorf("exeID is empty")
	}
	logger = logger.With("exeID", exeID)

	inputDataOptions := be.convertInputData(ctx)
	// logger.Debug("input data conversion complete", "inputDataOptions", inputDataOptions)

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
