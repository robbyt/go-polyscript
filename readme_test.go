package polyscript_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadmeQuickStart(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	script := `
		// The ctx object from the Go inputData map
		name := ctx.get("name")

		p := "."
		if ctx.get("excited") {
			p = "!"
		}

		message := "Hello, " + name + p

		// Return a map with our result
		{
			"greeting": message,
			"length": len(message)
		}
	`

	inputData := map[string]any{"name": "World"}

	evaluator, err := polyscript.FromRisorStringWithData(
		script,
		inputData,
		logger.Handler(),
	)
	require.NoError(t, err, "Should create evaluator successfully")

	ctx := context.Background()
	result, err := evaluator.Eval(ctx)
	require.NoError(t, err, "Should evaluate successfully")
	require.NotNil(t, result, "Result should not be nil")

	resultMap, ok := result.Interface().(map[string]any)
	require.True(t, ok, "Result should be a map")
	assert.Equal(t, "Hello, World.", resultMap["greeting"], "Greeting should match")
	assert.Equal(t, int64(13), resultMap["length"], "Length should be 13")
}

func TestReadmeStaticProvider(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	script := `
		name := ctx.get("name")
		excited := ctx.get("excited")

		p := "."
		if excited {
			p = "!"
		}

		message := "Hello, " + name + p

		{
			"greeting": message
		}
	`

	inputData := map[string]any{"name": "cats", "excited": true}
	evaluator, err := polyscript.FromRisorStringWithData(script, inputData, logger.Handler())
	require.NoError(t, err, "Should create evaluator successfully")

	ctx := context.Background()
	result, err := evaluator.Eval(ctx)
	require.NoError(t, err, "Should evaluate successfully")

	resultMap, ok := result.Interface().(map[string]any)
	require.True(t, ok, "Result should be a map")
	assert.Equal(t, "Hello, cats!", resultMap["greeting"], "Greeting should match with excitement")
}

func TestReadmeContextProvider(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	script := `
		name := ctx.get("name")
		relationship := ctx.get("relationship")

		{
			"name": name,
			"is_not_my_lover": relationship == false
		}
	`

	evaluator, err := polyscript.FromRisorString(script, logger.Handler())
	require.NoError(t, err, "Should create evaluator successfully")

	ctx := context.Background()
	runtimeData := map[string]any{"name": "Billie Jean", "relationship": false}
	enrichedCtx, err := evaluator.AddDataToContext(ctx, runtimeData)
	require.NoError(t, err, "Should add data to context successfully")

	result, err := evaluator.Eval(enrichedCtx)
	require.NoError(t, err, "Should evaluate successfully")

	resultMap, ok := result.Interface().(map[string]any)
	require.True(t, ok, "Result should be a map")
	assert.Equal(t, "Billie Jean", resultMap["name"], "Name should match")
	assert.Equal(t, true, resultMap["is_not_my_lover"], "Relationship status should be correct")
}

func TestReadmeCombiningStaticAndDynamic(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	script := `
		// Access both static and dynamic data
		name := ctx.get("name")
		excited := ctx.get("excited")

		p := "."
		if excited {
			p = "!"
		}

		message := "Hello, " + name + p

		{
			"greeting": message
		}
	`

	staticData := map[string]any{
		"name":    "User",
		"excited": true,
	}

	evaluator, err := polyscript.FromRisorStringWithData(script, staticData, logger.Handler())
	require.NoError(t, err, "Should create evaluator with static data")

	requestData := map[string]any{"name": "Robert"}
	enrichedCtx, err := evaluator.AddDataToContext(context.Background(), requestData)
	require.NoError(t, err, "Should add runtime data to context")

	result, err := evaluator.Eval(enrichedCtx)
	require.NoError(t, err, "Should evaluate with combined data")

	resultMap, ok := result.Interface().(map[string]any)
	require.True(t, ok, "Result should be a map")
	assert.Equal(t, "Hello, Robert!", resultMap["greeting"], "Should use runtime name over static")
}

func TestReadmeStarlark(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	scriptContent := `
# Starlark has access to ctx variable
name = ctx["name"]
message = "Hello, " + name + "!"

# Create the result dictionary
result = {"greeting": message, "length": len(message)}

# Assign to _ to return the value
_ = result
`

	staticData := map[string]any{"name": "World"}
	evaluator, err := polyscript.FromStarlarkStringWithData(
		scriptContent,
		staticData,
		logger.Handler(),
	)
	require.NoError(t, err, "Should create Starlark evaluator")

	result, err := evaluator.Eval(context.Background())
	require.NoError(t, err, "Should evaluate Starlark script")

	resultMap, ok := result.Interface().(map[string]any)
	require.True(t, ok, "Result should be a map")
	assert.Equal(t, "Hello, World!", resultMap["greeting"], "Greeting should match")
	assert.Equal(t, int64(13), resultMap["length"], "Length should be 13")
}

func TestReadmeExtism(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	staticData := map[string]any{"input": "World"}
	evaluator, err := polyscript.FromExtismBytesWithData(
		wasmdata.TestModule,
		staticData,
		logger.Handler(),
		wasmdata.EntrypointGreet,
	)
	require.NoError(t, err, "Should create Extism evaluator")

	result, err := evaluator.Eval(context.Background())
	require.NoError(t, err, "Should evaluate WASM module")
	require.NotNil(t, result, "Result should not be nil")

	resultMap, ok := result.Interface().(map[string]any)
	require.True(t, ok, "Result should be a map")
	assert.Contains(t, resultMap, "greeting", "Result should contain greeting field")
	assert.Equal(t, "Hello, World!", resultMap["greeting"], "Greeting should match")
}
