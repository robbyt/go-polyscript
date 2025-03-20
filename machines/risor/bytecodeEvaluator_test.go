package risor

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/internal/helpers"
)

var emptyScriptData = make(map[string]any)

// TestValidScript tests evaluating valid scripts
func TestValidScript(t *testing.T) {
	t.Parallel()
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(handler)) // Set as the global default logger

	// Define the Risor script
	scriptContent := `
func handle(request) {
	if request == nil {
		return error("request is nil")
	}
	if request["Method"] == "POST" {
		return "post"
	}
	if request["URL_Path"] == "/hello" {
		return true
	}
	return false
}
print(ctx)
handle(ctx["request"])
`
	loader, err := loader.NewFromString(scriptContent)
	require.NoError(t, err, "Failed to create new loader")

	opt := &RisorOptions{Globals: []string{constants.Ctx}}
	exe, err := script.NewExecutableUnit(
		handler,
		scriptContent,
		loader,
		NewCompiler(handler, opt),
		emptyScriptData,
	)
	require.NoError(t, err, "Failed to create new version")

	evaluator := NewBytecodeEvaluator(handler, nil)
	require.NotNil(t, evaluator, "BytecodeEvaluator should not be nil")

	t.Run("get request", func(t *testing.T) {
		// Create the HttpRequest data object
		req := httptest.NewRequest("GET", "/hello", nil)
		rMap, err := helpers.RequestToMap(req)
		require.NoError(t, err, "Failed to create HttpRequest data object")
		require.NotNil(t, rMap, "HttpRequest data should not be nil")
		require.Equal(t, "/hello", rMap["URL_Path"], "Expected request URL path to be /hello")

		evalData := map[string]any{
			constants.Request: rMap,
		}

		// ctx["eval_data"] => evalData
		//nolint:staticcheck // Temporarily ignoring the "string as context key" warning until type system is fixed
		ctx := context.WithValue(context.Background(), constants.EvalData, evalData)

		// Evaluate the script with the provided HttpRequest
		response, err := evaluator.Eval(ctx, exe)
		require.NoError(t, err, "Did not expect an error but got one")
		require.NotNil(t, response, "Response should not be nil")

		// Assert the type and value of the response
		require.Equal(t, data.Types("bool"), response.Type(), "Expected response type to be bool")
		require.Equal(t, "true", response.Inspect(), "Inspect() should return 'true'")

		// Assert the value of the response
		boolValue, ok := response.Interface().(bool)
		require.True(t, ok, "Expected response value to be a bool")
		require.True(t, boolValue, "Expected response value to be true")
	})

	t.Run("get request with a different path", func(t *testing.T) {
		// Create the HttpRequest data object
		req := httptest.NewRequest("GET", "/world", nil)
		rMap, err := helpers.RequestToMap(req)
		require.NoError(t, err, "Failed to create HttpRequest data object")
		require.NotNil(t, rMap, "HttpRequest data should not be nil")
		require.Equal(t, "/world", rMap["URL_Path"], "Expected request URL path to be /world")

		evalData := make(map[string]any)
		evalData[constants.Request] = rMap

		// ctx["eval_data"] => evalData
		//nolint:staticcheck // Temporarily ignoring the "string as context key" warning until type system is fixed
		ctx := context.WithValue(context.Background(), constants.EvalData, evalData)

		// Evaluate the script with the provided HttpRequest
		response, err := evaluator.Eval(ctx, exe)
		require.NoError(t, err, "Did not expect an error but got one")
		require.NotNil(t, response, "Response should not be nil")

		// Assert the type and value of the response
		require.Equal(t, data.Types("bool"), response.Type(), "Expected response type to be bool")
		require.Equal(t, "false", response.Inspect())

		// Assert the value of the response
		boolValue, ok := response.Interface().(bool)
		require.True(t, ok, "Expected response value to be a bool")
		require.False(t, boolValue, "Expected response value to be false")
	})

	t.Run("post request", func(t *testing.T) {
		// Create the HttpRequest data object
		req := httptest.NewRequest("POST", "/hello", nil)
		rMap, err := helpers.RequestToMap(req)
		require.NoError(t, err, "Failed to create HttpRequest data object")
		require.NotNil(t, rMap, "HttpRequest data should not be nil")
		require.Equal(t, "/hello", rMap["URL_Path"], "Expected request URL path to be /hello")

		evalData := make(map[string]any)
		evalData[constants.Request] = rMap

		//nolint:staticcheck // Temporarily ignoring the "string as context key" warning until type system is fixed
		ctx := context.WithValue(context.Background(), constants.EvalData, evalData)

		// Evaluate the script with the provided HttpRequest
		response, err := evaluator.Eval(ctx, exe)
		require.NoError(t, err, "Did not expect an error but got one")
		require.NotNil(t, response, "Response should not be nil")

		// Assert the type and value of the response
		require.Equal(t, data.Types("string"), response.Type())
		require.Equal(t, "\"post\"", response.Inspect())

		// Assert the value of the response
		strValue, ok := response.Interface().(string)
		require.True(t, ok, "Expected response value to be a string")
		require.Equal(t, strValue, "post")
	})
}
