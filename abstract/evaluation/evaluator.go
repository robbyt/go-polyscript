package evaluation

import (
	"context"
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

// DataPreparer prepares data for script evaluation by enriching a context.
// This interface supports separating data preparation from evaluation, enabling
// distributed architectures where these steps can occur on different systems.
type DataPreparer interface {
	// PrepareContext enriches a context with data for script evaluation.
	// It processes input data according to the machine implementation and stores it
	// in the context using the ExecutableUnit's DataProvider.
	//
	// The variadic data parameter accepts HTTP requests, maps, structs, and other types
	// that are converted appropriately for the script engine.
	//
	// Example:
	//  scriptData := map[string]any{"greeting": "Hello, World!"}
	//  enrichedCtx, err := evaluator.PrepareContext(ctx, request, scriptData)
	//  if err != nil {
	//      return err
	//  }
	//  result, err := evaluator.Eval(enrichedCtx)
	PrepareContext(ctx context.Context, data ...any) (context.Context, error)
}

// Evaluator combines the EvalOnly and EvalDataPreparer interfaces,
// providing a unified API for data preparation and script evaluation.
// It allows these steps to be performed separately while maintaining their
// logical connection, supporting distributed processing architectures.
type Evaluator interface {
	EvalOnly
	DataPreparer
}
