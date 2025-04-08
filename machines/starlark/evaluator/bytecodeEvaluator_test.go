package evaluator

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/machines/starlark/compiler"
	"github.com/stretchr/testify/require"
)

// TestValidScript tests evaluating valid scripts
func TestValidScript(t *testing.T) {
	t.Parallel()

	// Defines a Starlark script that can handle HTTP requests
	scriptContent := `
def request_handler(request):
    if request == None:
        fail("request is None")
    if request["Method"] == "POST":
        return "post"
    if request["URL_Path"] == "/hello":
        return True
    return False

print(ctx)
_ = request_handler(ctx.get("request"))
`

	evalBuilder := func(t *testing.T, scriptContent string) (*script.ExecutableUnit, *BytecodeEvaluator) {
		t.Helper()
		loader, err := loader.NewFromString(scriptContent)
		require.NoError(t, err, "Failed to create new loader")

		// Create test logger
		handler := slog.NewTextHandler(os.Stdout, nil)

		// Create a context provider to use with our test context
		ctxProvider := data.NewContextProvider(constants.EvalData)

		// Create compiler with options
		compiler, err := compiler.NewCompiler(
			compiler.WithLogHandler(handler),
			compiler.WithCtxGlobal(),
		)
		require.NoError(t, err, "Failed to create compiler")

		exe, err := script.NewExecutableUnit(
			handler,
			scriptContent,
			loader,
			compiler,
			ctxProvider,
		)
		require.NoError(t, err, "Failed to create new version")

		evaluator := NewBytecodeEvaluator(handler, exe)
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
				name:           "Handles /hello",
				script:         scriptContent,
				expected:       "True",
				expectedObject: true,
				urlPath:        "/hello",
			},
			{
				name:           "Handles other paths",
				script:         scriptContent,
				expected:       "False",
				expectedObject: false,
				urlPath:        "/other",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Setup the test
				exe, evaluator := evalBuilder(t, tt.script)
				_ = exe // We no longer need to pass this to Eval

				// Create the HttpRequest data object
				req := httptest.NewRequest("GET", tt.urlPath, nil)
				rMap, err := helpers.RequestToMap(req)
				require.NoError(t, err, "Failed to create HttpRequest data object")

				evalData := map[string]any{
					constants.Request: rMap,
				}

				ctx := context.WithValue(context.Background(), constants.EvalData, evalData)

				// Evaluate the script with the provided HttpRequest
				response, err := evaluator.Eval(ctx)
				require.NoError(t, err, "Did not expect an error")
				require.NotNil(t, response, "Response should not be nil")

				// Assert the string representation of the response
				require.Equal(t, tt.expected, response.Inspect())

				// Assert the actual value of the response
				require.Equal(t, tt.expectedObject, response.Interface())
			})
		}
	})

	t.Run("post request", func(t *testing.T) {
		// Setup the test
		exe, evaluator := evalBuilder(t, scriptContent)
		_ = exe // We no longer need to pass this to Eval

		// Create the HttpRequest data object
		req := httptest.NewRequest("POST", "/hello", nil)
		rMap, err := helpers.RequestToMap(req)
		require.NoError(t, err, "Failed to create HttpRequest data object")

		evalData := map[string]any{
			constants.Request: rMap,
		}

		ctx := context.WithValue(context.Background(), constants.EvalData, evalData)

		// Evaluate the script with the provided HttpRequest
		response, err := evaluator.Eval(ctx)
		require.NoError(t, err, "Did not expect an error")
		require.NotNil(t, response, "Response should not be nil")

		// Assert the string representation of the response
		require.Equal(t, "\"post\"", response.Inspect())

		// Assert the actual value of the response
		require.Equal(t, "post", response.Interface())
	})
}
