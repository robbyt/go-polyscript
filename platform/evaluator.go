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

	// Close releases any resources held by the compiled script. It is a
	// no-op for implementations that hold nothing to release. In one that
	// does, Eval returns an error as soon as Close has been called. Close
	// waits for in-flight Evals only until ctx is done: if they return first,
	// it releases the resources and returns the release error; if ctx is done
	// first, it returns an error wrapping ctx's error and the resources are
	// released when the last in-flight Eval returns. With nothing in flight
	// it releases them even if ctx is already done.
	//
	// Like [net/http.Server.Shutdown], Close is safe to call more than once
	// and concurrently: the resources are released once, and each call's ctx
	// bounds only that call's wait.
	//
	// An evaluator also closes itself this way when its lifetime ctx is done.
	// That is the ctx passed to its constructor, unless a lifetime option
	// supplied another. Call Close or let the lifetime end: an evaluator
	// that is neither closed nor bound to a lifetime that ends keeps its
	// resources, since nothing releases them when it is garbage collected.
	//
	// Since Close may wait for in-flight Evals, calling it from within an
	// Eval, such as by a data provider, blocks until ctx is done. Each engine
	// documents its own behaviour.
	Close(ctx context.Context) error
}
