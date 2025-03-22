package engine

import (
	"context"
)

// EvalDataPreparer prepares data for script evaluation by enriching a context.
// This interface supports separating data preparation from evaluation, enabling
// distributed architectures where these steps can occur on different systems.
type EvalDataPreparer interface {
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

// EvaluatorWithPrep combines the Evaluator and EvalDataPreparer interfaces,
// providing a unified API for data preparation and script evaluation.
// It allows these steps to be performed separately while maintaining their
// logical connection, supporting distributed processing architectures.
type EvaluatorWithPrep interface {
	Evaluator
	EvalDataPreparer
}
