package snippets

import (
	"context"

	"github.com/robbyt/go-polyscript"
)

// godocNew is the first quick start fragment in the polyscript package doc.
func godocNew(ctx context.Context, script string) error {
	// snippet:begin new
	eval, err := polyscript.New[polyscript.Risor](ctx, polyscript.FromString(script))
	if err != nil {
		return err
	}
	result, err := eval.Eval(ctx)
	// snippet:end
	return printResult(result, err)
}

// godocStaticData is the static data fragment in the polyscript package doc.
func godocStaticData(ctx context.Context, script string) error {
	// snippet:begin static-data
	eval, err := polyscript.New[polyscript.Risor](
		ctx,
		polyscript.FromString(script),
		polyscript.WithStaticData[polyscript.Risor](map[string]any{"name": "World"}),
	)
	// snippet:end
	return evalAndPrint(ctx, eval, err)
}

// godocExtism is the Extism fragment in the polyscript package doc.
func godocExtism(ctx context.Context, wasmBytes []byte) error {
	// snippet:begin extism
	eval, err := polyscript.New[polyscript.Extism](
		ctx,
		polyscript.FromBytes(wasmBytes),
		polyscript.WithEntryPoint("greet"),
	)
	if err != nil {
		return err
	}
	defer eval.Close(ctx)
	// snippet:end
	enrichedCtx, err := eval.AddDataToContext(ctx, map[string]any{"input": "World"})
	return evalAndPrint(enrichedCtx, eval, err)
}

// godocTypeArgument is the type argument fragment in the polyscript package
// doc.
func godocTypeArgument(ctx context.Context, script string, data map[string]any) error {
	// snippet:begin type-argument
	opt := polyscript.WithStaticData[polyscript.Risor](data)
	// snippet:end
	eval, err := polyscript.New[polyscript.Risor](ctx, polyscript.FromString(script), opt)
	return evalAndPrint(ctx, eval, err)
}

// godocConstructorRisor is the Risor fragment in the polyscript.New doc.
func godocConstructorRisor(ctx context.Context, script string) error {
	// snippet:begin constructor-risor
	eval, err := polyscript.New[polyscript.Risor](ctx, polyscript.FromString(script))
	// snippet:end
	return evalAndPrint(ctx, eval, err)
}

// godocConstructorExtism is the Extism fragment in the polyscript.New doc.
func godocConstructorExtism(ctx context.Context, wasmBytes []byte) error {
	// snippet:begin constructor-extism
	eval, err := polyscript.New[polyscript.Extism](
		ctx,
		polyscript.FromBytes(wasmBytes),
		polyscript.WithEntryPoint("greet"),
		polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "World"}),
	)
	// snippet:end
	return evalAndPrint(ctx, eval, err)
}
