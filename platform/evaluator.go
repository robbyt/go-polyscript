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

	// Close releases resources held by the compiled script, such as the
	// Extism engine's wazero runtime. It is safe to call more than once.
	// For engines without releasable resources (Risor, Starlark) Close is a
	// no-op. On engines that hold resources (Extism), Eval returns an error
	// once Close is called, without calling the data provider, while Close
	// itself waits for in-flight Evals, including their data loading. Close's
	// ctx cannot interrupt that wait; cancelling an Eval's ctx shortens it
	// only if the Eval honours cancellation, both in its data provider and,
	// for Extism, in the guest (the default unless ctx termination is
	// disabled). A data provider must not call Close or Eval on the same
	// evaluator.
	Close(ctx context.Context) error
}
