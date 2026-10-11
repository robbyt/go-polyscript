package snippets

import (
	"context"

	"github.com/robbyt/go-polyscript"
)

// fromFile is the README "From a file on disk" fragment.
func fromFile(scriptPath string) error {
	// snippet:begin from-file
	ctx := context.Background()

	// scriptPath is an absolute path, such as "/etc/polyscript/hello.risor"
	evaluator, err := polyscript.New[polyscript.Risor](
		ctx,
		polyscript.FromFile(scriptPath),
	)
	// snippet:end
	return evalAndPrint(ctx, evaluator, err)
}
