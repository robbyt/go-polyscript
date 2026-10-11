package snippets

// snippet:begin direct-extism
import (
	"context"

	"github.com/robbyt/go-polyscript/engines/extism"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

// snippet:end

// directExtism is the README "Direct engine subpackage usage" fragment for
// Extism.
func directExtism(wasmBytes []byte) error {
	// snippet:begin direct-extism
	ctx := context.Background()

	wasmLdr, err := loader.NewFromBytes(wasmBytes) // wasmBytes is a compiled WASM module
	if err != nil {
		return err
	}

	eval, err := extism.FromExtismLoader(
		ctx,
		wasmLdr,
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
