package engine

import (
	"context"
)

// Evaluator is the interface for the generic code evaluator.
type Evaluator interface {
	// Eval evaluates the pre-compiled script with data from the context.
	// The script and its configuration were provided during evaluator creation.
	// Runtime data is retrieved using the ExecutableUnit's DataProvider.
	//
	// This design encourages the "compile once, run many times" pattern,
	// where script compilation (expensive) is separated from execution (inexpensive).
	// For dynamic data, use a ContextProvider with the constants.EvalData key.
	Eval(ctx context.Context) (EvaluatorResponse, error)
}
