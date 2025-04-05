package polyscript_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/extism"
	"github.com/robbyt/go-polyscript/machines/risor"
	"github.com/robbyt/go-polyscript/machines/starlark"
	"github.com/robbyt/go-polyscript/options"
	"github.com/stretchr/testify/assert"
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

func TestWithCompositeProvider(t *testing.T) {
	// Create a simple script that uses composite data
	script := `print(ctx["static_key"], ", ", ctx["input_data"]["dynamic_key"])`

	// Create static data
	staticData := map[string]any{
		"static_key": "static_value",
	}

	// Create an evaluator with composite provider
	evaluator, err := polyscript.FromStarlarkString(
		script,
		polyscript.WithCompositeProvider(staticData),
		starlark.WithGlobals([]string{constants.Ctx}),
	)
	require.NoError(t, err)
	require.NotNil(t, evaluator)

	// Test adding dynamic data
	ctx := context.Background()
	dynamicData := map[string]any{"dynamic_key": "dynamic_value"}
	enrichedCtx, err := evaluator.PrepareContext(ctx, dynamicData)
	require.NoError(t, err)

	// Execute the script (won't fail if print works correctly)
	_, err = evaluator.Eval(enrichedCtx)
	require.NoError(t, err)
}

func TestPrepareAndEval(t *testing.T) {
	// Create a simple Risor evaluator
	script := `
		name := ctx["input_data"]["name"]
		{
			"message": "Hello, " + name + "!",
			"length": len(name)
		}
	`

	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create an evaluator with the CompositeProvider
	evaluator, err := polyscript.FromRisorString(
		script,
		options.WithDefaults(),
		options.WithLogger(logger.Handler()),
		polyscript.WithCompositeProvider(map[string]any{}),
		risor.WithGlobals([]string{constants.Ctx}),
	)
	require.NoError(t, err)

	// Test the PrepareAndEval function
	result, err := polyscript.PrepareAndEval(
		context.Background(),
		evaluator,
		map[string]any{"name": "World"},
	)
	require.NoError(t, err)

	// Verify the result
	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Hello, World!", resultMap["message"])

	// Check length without assuming the exact numeric type
	length := resultMap["length"]
	require.NotNil(t, length, "length field should be present")
	switch v := length.(type) {
	case int64:
		assert.Equal(t, int64(5), v, "length should be 5")
	case float64:
		assert.Equal(t, float64(5), v, "length should be 5")
	default:
		t.Errorf("length is unexpected type %T", v)
	}
}

func TestEvalAndExtractMap(t *testing.T) {
	// Create a simple Risor evaluator
	script := `
		{
			"message": "Hello, Static!",
			"length": 12
		}
	`

	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create an evaluator
	evaluator, err := polyscript.FromRisorString(
		script,
		options.WithDefaults(),
		options.WithLogger(logger.Handler()),
		risor.WithGlobals([]string{constants.Ctx}),
	)
	require.NoError(t, err)

	// Test EvalAndExtractMap
	resultMap, err := polyscript.EvalAndExtractMap(context.Background(), evaluator)
	require.NoError(t, err)

	// Verify the result
	assert.Equal(t, "Hello, Static!", resultMap["message"])

	// Check length without assuming the exact numeric type
	length := resultMap["length"]
	require.NotNil(t, length, "length field should be present")
	switch v := length.(type) {
	case int64:
		assert.Equal(t, int64(12), v, "length should be 12")
	case float64:
		assert.Equal(t, float64(12), v, "length should be 12")
	default:
		t.Errorf("length is unexpected type %T", v)
	}

	// Test with nil result
	nilScript := `nil`
	nilEvaluator, err := polyscript.FromRisorString(
		nilScript,
		options.WithDefaults(),
		risor.WithGlobals([]string{constants.Ctx}),
	)
	require.NoError(t, err)

	nilResult, err := polyscript.EvalAndExtractMap(context.Background(), nilEvaluator)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{}, nilResult)

	// Test with non-map result (should error)
	numScript := `42`
	numEvaluator, err := polyscript.FromRisorString(
		numScript,
		options.WithDefaults(),
		risor.WithGlobals([]string{constants.Ctx}),
	)
	require.NoError(t, err)

	_, err = polyscript.EvalAndExtractMap(context.Background(), numEvaluator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "result is not a map")
}

func TestNewWithData(t *testing.T) {
	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test data
	staticData := map[string]any{
		"app_version": "1.0.0",
		"config": map[string]any{
			"timeout": 30,
		},
	}

	// Test NewRisorWithData
	risorScript := `
		// Access static data
		version := ctx["app_version"]
		timeout := ctx["config"]["timeout"]
		
		// Access dynamic data
		name := ctx["input_data"]["name"]
		
		{
			"message": "Hello, " + name + " (v" + version + ")",
			"timeout": timeout
		}
	`

	risorEval, err := polyscript.NewRisorWithData(risorScript, staticData, logger.Handler())
	require.NoError(t, err)

	// Test with dynamic data
	ctx := context.Background()
	dynamicData := map[string]any{"name": "Risor User"}
	enrichedCtx, err := risorEval.PrepareContext(ctx, dynamicData)
	require.NoError(t, err)

	risorResult, err := risorEval.Eval(enrichedCtx)
	require.NoError(t, err)

	risorMap, ok := risorResult.Interface().(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Hello, Risor User (v1.0.0)", risorMap["message"])

	// Check timeout without assuming specific number type
	timeout := risorMap["timeout"]
	require.NotNil(t, timeout, "timeout field should be present")
	switch v := timeout.(type) {
	case int64:
		assert.Equal(t, int64(30), v, "timeout should be 30")
	case float64:
		assert.Equal(t, float64(30), v, "timeout should be 30")
	default:
		t.Errorf("timeout is unexpected type %T", v)
	}

	// Test NewStarlarkWithData
	starlarkScript := "# Access static data\nversion = ctx[\"app_version\"]\ntimeout = ctx[\"config\"][\"timeout\"]\n\n# Access dynamic data\nname = ctx[\"input_data\"][\"name\"]\n\n# Return result\n= {\n    \"message\": \"Hello, \" + name + \" (v\" + version + \")\",\n    \"timeout\": timeout\n}\n\n# Starlark requires assignment to _ for return values\n_ = result"

	starlarkEval, err := polyscript.NewStarlarkWithData(
		starlarkScript,
		staticData,
		logger.Handler(),
	)
	require.NoError(t, err)

	// Test with dynamic data
	dynamicData = map[string]any{"name": "Starlark User"}
	enrichedCtx, err = starlarkEval.PrepareContext(ctx, dynamicData)
	require.NoError(t, err)

	starlarkResult, err := starlarkEval.Eval(enrichedCtx)
	require.NoError(t, err)

	starlarkMap, ok := starlarkResult.Interface().(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Hello, Starlark User (v1.0.0)", starlarkMap["message"])

	// Check timeout without assuming specific number type
	starlarkTimeout := starlarkMap["timeout"]
	require.NotNil(t, starlarkTimeout, "timeout field should be present")
	assert.Equal(t, int64(30), starlarkTimeout, "timeout should be 30")

	// Skip Extism test as it requires a WASM file
	// Testing NewExtismWithData would be similar but would need actual WASM
}
