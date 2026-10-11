package snippets

import (
	"context"
	"fmt"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/platform/data"
)

// staticProvider is the README "StaticProvider" fragment.
func staticProvider(script string) error {
	// snippet:begin static-provider
	ctx := context.Background()

	evaluator, err := polyscript.New[polyscript.Risor](
		ctx,
		polyscript.FromString(script),
		polyscript.WithStaticData[polyscript.Risor](map[string]any{"name": "cats", "excited": true}),
	)
	// snippet:end
	return evalAndPrint(ctx, evaluator, err)
}

// contextProvider is the README "ContextProvider" fragment.
func contextProvider(script string) error {
	// snippet:begin context-provider
	ctx := context.Background()

	evaluator, err := polyscript.New[polyscript.Risor](ctx, polyscript.FromString(script))
	if err != nil {
		return err
	}

	runtimeData := map[string]any{"name": "Billie Jean", "relationship": false}
	enrichedCtx, err := evaluator.AddDataToContext(ctx, runtimeData)
	if err != nil {
		return err
	}

	// Execute with the "enriched" context containing the link to the input data
	result, err := evaluator.Eval(enrichedCtx)
	// snippet:end
	return printResult(result, err)
}

// derivedContexts is the README "Concurrency and thread-safety" fragment.
func derivedContexts(
	parent context.Context,
	provider *data.ContextProvider,
	requestData, moreData map[string]any,
) error {
	// snippet:begin derived-contexts
	ctx, err := provider.AddDataToContext(parent, requestData)
	if err != nil {
		return err
	}
	ctx, err = provider.AddDataToContext(ctx, moreData) // build on the returned ctx
	if err != nil {
		return err
	}
	// snippet:end
	got, err := provider.GetData(ctx)
	if err != nil {
		return err
	}
	fmt.Println(got)
	return nil
}

// staticAndDynamic is the README "Combining Static and Dynamic Runtime Data"
// fragment.
func staticAndDynamic(script string) error {
	// snippet:begin static-and-dynamic
	ctx := context.Background()

	staticData := map[string]any{
		"name":    "User",
		"excited": true,
	}

	// Create the evaluator with the static data
	evaluator, err := polyscript.New[polyscript.Risor](
		ctx,
		polyscript.FromString(script),
		polyscript.WithStaticData[polyscript.Risor](staticData),
	)
	if err != nil {
		return err
	}

	// For each request, prepare dynamic data
	requestData := map[string]any{"name": "Robert"}
	enrichedCtx, err := evaluator.AddDataToContext(ctx, requestData)
	if err != nil {
		return err
	}

	// Execute with both static and dynamic data available
	result, err := evaluator.Eval(enrichedCtx)
	// snippet:end
	return printResult(result, err)
}
