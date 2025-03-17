package starlark

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/data/helpers"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/execution/script/loader"
)

var emptyScriptData = make(map[string]any)

// TestValidScript tests evaluating valid scripts
func TestValidScript(t *testing.T) {
	t.Parallel()
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(handler)) // Set as the global default logger

	reqPathScript := `
def request_handler(request):
    if request == None:
        return "request is nil"
    if request["Method"] == "POST":
        return "POST"
    if request["URL_Path"] == "/hello":
        return True
    return False

print(ctx)
_ = request_handler(ctx.get("request"))
`

	evalBuilder := func(t *testing.T, scriptContent string) (*script.ExecutableUnit, *BytecodeEvaluator) {
		loader, err := loader.NewFromString(scriptContent)
		require.NoError(t, err, "Failed to create new loader")

		// Create test logger
		handler := slog.NewTextHandler(os.Stdout, nil)

		exe, err := script.NewExecutableUnit(
			handler,
			scriptContent,
			loader,
			NewCompiler(handler, &BasicCompilerOptions{Globals: []string{constants.Ctx}}),
			emptyScriptData,
		)
		require.NoError(t, err, "Failed to create new version")

		evaluator := NewBytecodeEvaluator(handler)
		require.NotNil(t, evaluator, "BytecodeEvaluator should not be nil")

		return exe, evaluator
	}

	t.Run("get request", func(t *testing.T) {
		tests := []struct {
			name           string
			script         string
			expected       string
			expectedObject any
			urlPath        string
		}{
			{
				name:           "trueScript",
				script:         reqPathScript,
				expected:       "True",
				expectedObject: true,
				urlPath:        "/hello",
			},
			{
				name:           "falseScript",
				script:         reqPathScript,
				expected:       "False",
				expectedObject: false,
				urlPath:        "/notHello",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				version, evaluator := evalBuilder(t, tt.script)
				// Create the HttpRequest data object
				req := httptest.NewRequest("GET", tt.urlPath, nil)
				rMap, err := helpers.RequestToMap(req)
				require.NoError(t, err, "Failed to create HttpRequest data object")
				require.NotNil(t, rMap, "HttpRequest data should not be nil")
				require.Equal(t, tt.urlPath, rMap["URL_Path"], "Expected request URL path to be %s", tt.urlPath)

				evalData := make(map[string]any)
				evalData[constants.Request] = rMap

				// ctx["eval_data"] => evalData
				//nolint:staticcheck // Temporarily ignoring the "string as context key" warning until type system is fixed
				ctx := context.WithValue(context.Background(), constants.EvalData, evalData)

				// Evaluate the script with the provided HttpRequest
				response, err := evaluator.Eval(ctx, version)
				require.NoError(t, err, "Did not expect an error but got one")
				require.NotNil(t, response, "Response should not be nil")

				assert.Equal(t, tt.expected, response.Inspect(), "Inspect() should return %s", tt.expected)
				assert.IsType(t, tt.expectedObject, response.Interface(), "Expected response value to be a %T", tt.expectedObject)
				assert.Equal(t, tt.expectedObject, response.Interface(), "Expected response value to be %v", tt.expectedObject)
			})
		}
	})

	t.Run("post request", func(t *testing.T) {
		version, evaluator := evalBuilder(t, reqPathScript)
		// Create the HttpRequest data object
		req := httptest.NewRequest("POST", "/hello", nil)
		rMap, err := helpers.RequestToMap(req)
		require.NoError(t, err, "Failed to create HttpRequest data object")
		require.NotNil(t, rMap, "HttpRequest data should not be nil")
		require.Equal(t, "/hello", rMap["URL_Path"], "Expected request URL path to be /hello")

		evalData := make(map[string]any)
		evalData[constants.Request] = rMap

		// ctx["eval_data"] => evalData
		//nolint:staticcheck // Temporarily ignoring the "string as context key" warning until type system is fixed
		ctx := context.WithValue(context.Background(), constants.EvalData, evalData)

		// Evaluate the script with the provided HttpRequest
		response, err := evaluator.Eval(ctx, version)
		require.NoError(t, err, "Did not expect an error but got one")
		require.NotNil(t, response, "Response should not be nil")

		// Assert the type and value of the response
		require.Equal(t, data.Types("string"), response.Type(), "Expected response type to be bool")
		require.Equal(t, "\"POST\"", response.Inspect())

		// Assert the value of the response
		strValue, ok := response.Interface().(string)
		require.True(t, ok, "Expected response value to be a bool")
		require.Equal(t, strValue, "POST")
	})
}
