package snippets

import (
	"context"

	"github.com/robbyt/go-polyscript"
)

// starlarkEngine is the README "Starlark" fragment.
func starlarkEngine() error {
	// snippet:begin starlark
	scriptContent := `
# Starlark has access to ctx variable
name = ctx["name"]
message = "Hello, " + name + "!"

# Create the result dictionary
result = {"greeting": message, "length": len(message)}

# Assign to _ to return the value
_ = result
`

	ctx := context.Background()

	evaluator, err := polyscript.New[polyscript.Starlark](
		ctx,
		polyscript.FromString(scriptContent),
		polyscript.WithStaticData[polyscript.Starlark](map[string]any{"name": "World"}),
	)
	if err != nil {
		return err
	}

	// Execute with a context
	result, err := evaluator.Eval(ctx)
	// snippet:end
	return printResult(result, err)
}
