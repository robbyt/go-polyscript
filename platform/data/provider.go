package data

import (
	"context"
)

// Getter defines the interface for retrieving data from a context.
type Getter interface {
	GetData(ctx context.Context) (map[string]any, error)
}

// Setter prepares data for script evaluation by enriching a context.
// This interface supports separating data preparation from evaluation, enabling
// distributed architectures where these steps can occur on different systems.
type Setter interface {
	// AddDataToContext enriches a context with data for script evaluation.
	// It processes input data according to the engine implementation and stores it
	// in the context using the ExecutableUnit's DataProvider.
	//
	// The variadic data parameter accepts maps with string keys and arbitrary values.
	// HTTP requests, structs, and other types should be wrapped in maps with descriptive keys.
	//
	// Example:
	//  scriptData := map[string]any{"greeting": "Hello, World!"}
	//  enrichedCtx, err := evaluator.AddDataToContext(ctx, map[string]any{"request": request}, scriptData)
	//  if err != nil {
	//      return err
	//  }
	//  result, err := evaluator.Eval(enrichedCtx)
	AddDataToContext(ctx context.Context, data ...map[string]any) (context.Context, error)
}

// Provider defines the interface for accessing runtime data for script execution.
type Provider interface {
	// Getter retrieves associated data from a context during script eval.
	Getter

	// Setter enriches a context with a link to data, allowing the script
	// to access it using the ExecutableUnit's DataProvider.
	Setter
}
