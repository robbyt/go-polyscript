package polyscript

import (
	"context"
	"fmt"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/script"
)

// EvaluatorWrapper wraps a machine-specific evaluator and stores the ExecutableUnit.
// This allows callers to follow the "compile once, run many times" pattern.
// It implements both the Evaluator and EvalDataPreparer interfaces.
type EvaluatorWrapper struct {
	delegate engine.Evaluator
	execUnit *script.ExecutableUnit
}

// NewEvaluatorWrapper creates a new evaluator wrapper
func NewEvaluatorWrapper(
	delegateEvaluator engine.Evaluator,
	execUnit *script.ExecutableUnit,
) engine.EvaluatorWithPrep {
	return &EvaluatorWrapper{
		delegate: delegateEvaluator,
		execUnit: execUnit,
	}
}

// Eval implements the engine.Evaluator interface
// It delegates to the wrapped evaluator using the stored ExecutableUnit
func (e *EvaluatorWrapper) Eval(ctx context.Context) (engine.EvaluatorResponse, error) {
	return e.delegate.Eval(ctx)
}

// PrepareContext implements the engine.EvalDataPreparer interface by enriching
// the context with data for script evaluation. It delegates to the wrapped evaluator
// if it implements EvalDataPreparer, otherwise uses the ExecutableUnit's DataProvider directly.
func (e *EvaluatorWrapper) PrepareContext(
	ctx context.Context,
	data ...any,
) (context.Context, error) {
	// If the delegate implements EvalDataPreparer, use it
	if preparer, ok := e.delegate.(engine.EvalDataPreparer); ok {
		return preparer.PrepareContext(ctx, data...)
	}

	// Fallback implementation using the executable unit's data provider
	if e.execUnit == nil || e.execUnit.GetDataProvider() == nil {
		return ctx, fmt.Errorf("no data provider available")
	}

	return e.execUnit.GetDataProvider().AddDataToContext(ctx, data...)
}

// GetExecutableUnit returns the stored ExecutableUnit
// This is useful for examining or modifying the unit
func (e *EvaluatorWrapper) GetExecutableUnit() *script.ExecutableUnit {
	return e.execUnit
}
