package evaluator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/robbyt/go-polyscript/engines/starlark/internal"
	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script"
	starlarkLib "go.starlark.net/starlark"
)

// Evaluator is an abstraction layer for evaluating code on the Starlark engine
type Evaluator struct {
	// universe is the global variable map for the Starlark engine
	universe starlarkLib.StringDict

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
	handler, logger := helpers.SetupLogger(handler, "starlark", "Evaluator")

	// Get universe with standard modules
	universe := internal.StarlarkModules()

	// Add eval-time contextual globals
	universe[constants.Ctx] = starlarkLib.None
	universe[string(constants.EvalData)] = starlarkLib.None

	return &Evaluator{
		universe:   universe,
		execUnit:   execUnit,
		logHandler: handler,
		logger:     logger,
	}
}

func (be *Evaluator) String() string {
	return "starlark.Evaluator"
}

// getDataProvider returns the data provider from the executable unit, or nil if unavailable.
func (be *Evaluator) getDataProvider() data.Provider {
	if be.execUnit == nil {
		return nil
	}
	return be.execUnit.GetDataProvider()
}

// loadInputData retrieves input data using the data provider in the executable unit.
// Returns a map that will be used as input for the Starlark engine.
func (be *Evaluator) loadInputData(ctx context.Context) (map[string]any, error) {
	return data.LoadInputData(ctx, be.logHandler.WithGroup("Evaluator"), be.getDataProvider())
}

// prepareGlobals merges the universe and input globals into a single Starlark dictionary
func (be *Evaluator) prepareGlobals(
	inputGlobals starlarkLib.StringDict,
) starlarkLib.StringDict {
	// Pre-allocate with exact capacity needed
	mergedGlobals := make(starlarkLib.StringDict, len(be.universe)+len(inputGlobals))

	// Copy the pre-populated universe first
	maps.Copy(mergedGlobals, be.universe)

	// Then add execution-specific globals, which may override universe values
	maps.Copy(mergedGlobals, inputGlobals)

	return mergedGlobals
}

// newThread returns a Starlark thread whose print() output goes to logger
// and which is cancelled when ctx is done. The caller must call the returned
// stop func once the thread has finished, to release the ctx registration.
func newThread(ctx context.Context, logger *slog.Logger, name string) (*starlarkLib.Thread, func() bool) {
	thread := &starlarkLib.Thread{
		Name: name,
		Print: func(thread *starlarkLib.Thread, msg string) {
			logger.InfoContext(ctx, msg, "starlark-thread", thread.Name)
		},
	}

	// context.AfterFunc avoids a goroutine leak when ctx is never cancelled
	// (e.g., context.Background()).
	stop := context.AfterFunc(ctx, func() {
		thread.Cancel(ctx.Err().Error())
	})
	return thread, stop
}

// withCtxErr adds ctx.Err() to the chain of a script error when ctx is done,
// so callers can detect a cancelled or timed-out Eval with errors.Is. The
// Starlark error only carries the cancellation reason as text.
func withCtxErr(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("execution cancelled: %w (%w)", ctxErr, err)
	}
	return err
}

// exec executes the bytecode with the provided globals
func (be *Evaluator) exec(
	ctx context.Context,
	prog *starlarkLib.Program,
	globals starlarkLib.StringDict,
) (*execResult, error) {
	logger := be.logger.WithGroup("exec")
	startTime := time.Now()

	thread, stop := newThread(ctx, logger, "eval")
	defer stop()

	// Execute the program
	finalGlobals, err := prog.Init(thread, globals)
	execTime := time.Since(startTime)

	if err != nil {
		var evalErr *starlarkLib.EvalError
		errors.As(err, &evalErr)
		return nil, withCtxErr(ctx, &Error{
			Msg:     fmt.Sprintf("starlark execution error: %s", err),
			EvalErr: evalErr,
		})
	}

	// Get the main value from globals
	// The "_" key contains the last value evaluated in the Starlark script
	mainVal := finalGlobals["_"]
	if mainVal == nil {
		mainVal = starlarkLib.None
	}

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

// Eval evaluates the loaded bytecode and passes the provided data into the Starlark engine
func (be *Evaluator) Eval(ctx context.Context) (platform.EvaluatorResponse, error) {
	logger := be.logger.WithGroup("Eval")
	if be.execUnit == nil {
		return nil, fmt.Errorf("executable unit is nil")
	}

	if be.execUnit.GetContent() == nil {
		return nil, fmt.Errorf("content is nil")
	}

	// Get bytecode from executable unit
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

	// 1. Type assert to Starlark program
	prog, ok := bytecode.(*starlarkLib.Program)
	if !ok {
		return nil, fmt.Errorf(
			"invalid bytecode type: expected *starlark.Program, got %T",
			bytecode,
		)
	}

	// 2. Get the raw input data
	rawInputData, err := be.loadInputData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get input data: %w", err)
	}

	// 3. Convert input data to Starlark values
	input, err := internal.ConvertToStarlarkFormat(rawInputData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input data: %w", err)
	}
	// Prepare globals by merging input with "universe"
	runtimeData := be.prepareGlobals(input)

	// 4. Execute the program
	result, err := be.exec(ctx, prog, runtimeData)
	if err != nil {
		return nil, fmt.Errorf("exec error: %w", err)
	}
	logger.DebugContext(ctx, "exec complete", "result", result)

	// 5. Collect results
	result.scriptExeID = exeID

	// Handle specific return types
	if result.Value == nil {
		logger.Warn("result value is nil")
		return result, nil
	}

	// Starlark has no top-level expression statement, so a script body conventionally
	// ends in a `def` block; auto-invoke with no args matches the "run my main()"
	// idiom. Diverges from Risor by design — see engines/README.md "Script Return
	// Value Handling".
	if callable, ok := result.Value.(starlarkLib.Callable); ok {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("execution cancelled: %w", err)
		}
		thread, stop := newThread(ctx, be.logger.WithGroup("exec"), "func")
		defer stop()
		callStart := time.Now()
		val, err := starlarkLib.Call(thread, callable, nil, nil)
		execTime := result.execTime + time.Since(callStart)
		if err != nil {
			var evalErr *starlarkLib.EvalError
			errors.As(err, &evalErr)
			return nil, withCtxErr(ctx, &Error{
				Msg:     fmt.Sprintf("error calling function: %s", err),
				EvalErr: evalErr,
			})
		}
		// "Freeze" the value to prevent any further modifications
		val.Freeze()
		return newEvalResult(be.logHandler, val, execTime, exeID), nil
	}

	return result, nil
}

// AddDataToContext implements the data.Setter interface which stores and prepares runtime data
// which can be eventually passed to the Eval method.
func (be *Evaluator) AddDataToContext(
	ctx context.Context,
	d map[string]any,
) (context.Context, error) {
	return data.AddDataToContextFromProvider(ctx, be.logHandler.WithGroup("Evaluator"), be.getDataProvider(), d)
}
