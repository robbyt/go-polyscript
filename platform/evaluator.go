package platform

import (
	"context"

	"github.com/robbyt/go-polyscript/platform/data"
)

// EvalOnly is the interface for the generic code evaluator.
type EvalOnly interface {
	// Eval evaluates the pre-compiled script with data from the context.
	// The script and its configuration were provided during evaluator creation.
	// Runtime data is retrieved using the ExecutableUnit's DataProvider.
	//
	// This design encourages the "compile once, run many times" pattern,
	// where script compilation (expensive) is separated from execution (inexpensive).
	// For dynamic data, use a ContextProvider with the constants.EvalData key.
	Eval(ctx context.Context) (EvaluatorResponse, error)
}

// Evaluator combines the EvalOnly and EvalDataPreparer interfaces,
// providing a unified API for data preparation and script evaluation.
// It allows these steps to be performed separately while maintaining their
// logical connection, supporting distributed processing architectures.
type Evaluator interface {
	EvalOnly
	data.Setter
}
