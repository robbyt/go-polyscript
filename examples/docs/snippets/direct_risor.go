package snippets

// snippet:begin direct-risor
import (
	"context"

	"github.com/robbyt/go-polyscript/engines/risor"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

// snippet:end

// directRisor is the README "Direct engine subpackage usage" fragment for
// Risor.
func directRisor() error {
	// snippet:begin direct-risor
	ctx := context.Background()

	ldr, err := loader.NewFromString(`{"name": ctx["name"]}`)
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
