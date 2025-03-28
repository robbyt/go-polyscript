package polyscript_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/extism"
	"github.com/robbyt/go-polyscript/machines/risor"
	"github.com/robbyt/go-polyscript/machines/starlark"
	"github.com/robbyt/go-polyscript/options"
	"github.com/stretchr/testify/require"
)

func TestNewStarlarkEvaluator(t *testing.T) {
	// Create a simple Starlark script
	script := `print("Hello, World!")`

	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, nil)

	// Create an evaluator with string loader
	evaluator, err := polyscript.FromStarlarkString(script,
		options.WithLogger(handler),
		starlark.WithGlobals([]string{"ctx"}),
	)

	require.NoError(t, err)
	require.NotNil(t, evaluator)
}

func TestNewRisorEvaluator(t *testing.T) {
	// Create a simple Risor script
	script := `print("Hello, World!")`

	// Create a logger handler
	handler := slog.NewTextHandler(os.Stdout, nil)

	// Create an evaluator with string loader
	evaluator, err := polyscript.FromRisorString(script,
		options.WithLogger(handler),
		risor.WithGlobals([]string{"ctx"}),
	)

	require.NoError(t, err)
	require.NotNil(t, evaluator)
}

func TestNewEvaluatorWithDataProvider(t *testing.T) {
	// Create a simple script that uses data
	script := `print(ctx["key"])`

	// Create a data provider
	provider := data.NewStaticProvider(map[string]any{
		"key": "value",
	})

	// Create an evaluator with string loader and data provider
	evaluator, err := polyscript.FromStarlarkString(script,
		options.WithDataProvider(provider),
		starlark.WithGlobals([]string{"ctx"}),
	)

	require.NoError(t, err)
	require.NotNil(t, evaluator)
}

func TestInvalidOptions(t *testing.T) {
	// Test with wrong machine type
	// Trying to use Starlark globals with Risor
	script := `print("Hello, World!")`

	// Create an evaluator with string loader
	_, err := polyscript.FromRisorString(script,
		starlark.WithGlobals([]string{"ctx"}), // This should fail because it's for Starlark
	)

	require.Error(t, err)

	// Trying to use Extism entry point with Starlark
	_, err = polyscript.FromStarlarkString(script,
		extism.WithEntryPoint("main"), // This should fail because it's for Extism
	)

	require.Error(t, err)
}

func TestWithNoLoader(t *testing.T) {
	// Test with no loader
	_, err := polyscript.NewStarlarkEvaluator()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no loader specified")
}

func TestFromStringWithError(t *testing.T) {
	// Empty script
	_, err := polyscript.FromStarlarkString("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "content is empty")
}

func TestFromExtismFile(t *testing.T) {
	// Skip this test if running outside the repo root
	t.Skip("Skipping test that requires absolute path to WASM file")

	// For the test to work, you'd need an absolute path to the WASM file
	// wasmPath := "/absolute/path/to/examples/extism/main.wasm"

	// // Create a logger handler
	// := slog.NewTextHandler(os.Stdout, nil)

	// // Create an evaluator with file loader
	// evaluator, err := polyscript.FromExtismFile(wasmPath,
	// 	options.WithLogger(handler),
	// 	extism.WithEntryPoint("main"),
	// )

	// require.NoError(t, err)
	// require.NotNil(t, evaluator)
}

func TestFromExtismFileWithError(t *testing.T) {
	// Test with non-existent file
	_, err := polyscript.FromExtismFile("non-existent-file.wasm")
	require.Error(t, err)
}
