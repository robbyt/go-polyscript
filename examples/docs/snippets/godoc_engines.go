package snippets

import (
	"context"
	"log/slog"

	"github.com/robbyt/go-polyscript/engines/extism"
	"github.com/robbyt/go-polyscript/engines/risor"
	"github.com/robbyt/go-polyscript/engines/starlark"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

// godocRisorQuickStart is the quick start fragment in the engines/risor
// package doc.
func godocRisorQuickStart(ctx context.Context) error {
	// snippet:begin risor-quick-start
	ldr, err := loader.NewFromString(`"hello"`)
	if err != nil {
		return err
	}

	eval, err := risor.FromRisorLoader(
		ctx,
		ldr,
		risor.WithStaticData(map[string]any{"name": "World"}),
	)
	if err != nil {
		return err
	}

	result, err := eval.Eval(ctx)
	// snippet:end
	return printResult(result, err)
}

// godocRisorOptions is the fragment in the risor.FromRisorLoader doc.
func godocRisorOptions(ctx context.Context, ldr loader.Loader) error {
	// snippet:begin risor-options
	eval, err := risor.FromRisorLoader(
		ctx,
		ldr,
		risor.WithLogHandler(slog.Default().Handler()),
		risor.WithStaticData(map[string]any{"version": "1.0.0"}),
	)
	// snippet:end
	return evalAndPrint(ctx, eval, err)
}

// godocStarlarkQuickStart is the quick start fragment in the engines/starlark
// package doc.
func godocStarlarkQuickStart(ctx context.Context) error {
	// snippet:begin starlark-quick-start
	ldr, err := loader.NewFromString(`_ = "hello"`)
	if err != nil {
		return err
	}

	eval, err := starlark.FromStarlarkLoader(
		ctx,
		ldr,
		starlark.WithStaticData(map[string]any{"name": "World"}),
	)
	if err != nil {
		return err
	}

	result, err := eval.Eval(ctx)
	// snippet:end
	return printResult(result, err)
}

// godocStarlarkOptions is the fragment in the starlark.FromStarlarkLoader
// doc.
func godocStarlarkOptions(ctx context.Context, ldr loader.Loader) error {
	// snippet:begin starlark-options
	eval, err := starlark.FromStarlarkLoader(
		ctx,
		ldr,
		starlark.WithLogHandler(slog.Default().Handler()),
		starlark.WithStaticData(map[string]any{"version": "1.0.0"}),
	)
	// snippet:end
	return evalAndPrint(ctx, eval, err)
}

// godocExtismQuickStart is the quick start fragment in the engines/extism
// package doc.
func godocExtismQuickStart(ctx context.Context, wasmBytes []byte) error {
	// snippet:begin extism-quick-start
	ldr, err := loader.NewFromBytes(wasmBytes)
	if err != nil {
		return err
	}

	eval, err := extism.FromExtismLoader(
		ctx,
		ldr,
		extism.WithEntryPoint("greet"),
		extism.WithStaticData(map[string]any{"input": "World"}),
	)
	if err != nil {
		return err
	}

	result, err := eval.Eval(ctx)
	// snippet:end
	return printResult(result, err)
}

// godocExtismOptions is the fragment in the extism.FromExtismLoader doc.
func godocExtismOptions(ctx context.Context, ldr loader.Loader) error {
	// snippet:begin extism-options
	eval, err := extism.FromExtismLoader(
		ctx,
		ldr,
		extism.WithEntryPoint("greet"),
		extism.WithLogHandler(slog.Default().Handler()),
		extism.WithStaticData(map[string]any{"input": "World"}),
	)
	// snippet:end
	return evalAndPrint(ctx, eval, err)
}
