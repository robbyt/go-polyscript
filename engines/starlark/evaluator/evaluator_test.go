package evaluator

import (
	"context"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/robbyt/go-polyscript/engines/starlark/compiler"
	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script"
	"github.com/robbyt/go-polyscript/platform/script/loader"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// evalBuilder is a helper function to create a test executor and evaluator
func evalBuilder(t *testing.T, scriptContent string) (*script.ExecutableUnit, *Evaluator) {
	t.Helper()
	loader, err := loader.NewFromString(scriptContent)
	require.NoError(t, err, "Failed to create new loader")

	// Create test logger
	handler := slog.NewTextHandler(os.Stdout, nil)

	// Create a context provider to use with our test context
	ctxProvider := data.NewContextProvider(constants.EvalData)

	// Create compiler with options
	compiler, err := compiler.New(
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

	evaluator := New(handler, exe)
	require.NotNil(t, evaluator, "Evaluator should not be nil")

	return exe, evaluator
}

// Mock the data.Provider interface
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) GetData(ctx context.Context) (map[string]any, error) {
	args := m.Called(ctx)
	if data, ok := args.Get(0).(map[string]any); ok {
		return data, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProvider) AddDataToContext(ctx context.Context, data ...any) (context.Context, error) {
	args := m.Called(ctx, data)
	if ctx, ok := args.Get(0).(context.Context); ok {
		return ctx, args.Error(1)
	}
	return ctx, args.Error(1)
}

// TestEvaluator_Evaluate tests evaluating starlark scripts
func TestEvaluator_Evaluate(t *testing.T) {
	t.Parallel()

	// Define a Starlark script that can handle HTTP requests
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

	t.Run("success cases", func(t *testing.T) {
		tests := []struct {
			name           string
			script         string
			requestMethod  string
			urlPath        string
			expected       string
			expectedObject any
		}{
			{
				name:           "GET request to /hello",
				script:         scriptContent,
				requestMethod:  "GET",
				urlPath:        "/hello",
				expected:       "True",
				expectedObject: true,
			},
			{
				name:           "GET request to other path",
				script:         scriptContent,
				requestMethod:  "GET",
				urlPath:        "/other",
				expected:       "False",
				expectedObject: false,
			},
			{
				name:           "POST request",
				script:         scriptContent,
				requestMethod:  "POST",
				urlPath:        "/hello",
				expected:       "\"post\"",
				expectedObject: "post",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Setup the test
				_, evaluator := evalBuilder(t, tt.script)

				// Create the HttpRequest data object
				req := httptest.NewRequest(tt.requestMethod, tt.urlPath, nil)
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

	t.Run("error cases", func(t *testing.T) {
		// Test nil executable unit
		t.Run("nil executable unit", func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			evaluator := New(handler, nil)

			response, err := evaluator.Eval(context.Background())
			require.Error(t, err)
			require.Nil(t, response)
			require.Contains(t, err.Error(), "executable unit is nil")
		})

		// Test content nil
		t.Run("content nil", func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			exe := &script.ExecutableUnit{
				ID:      "test-nil-content",
				Content: nil, // Deliberately nil content
			}
			evaluator := New(handler, exe)

			response, err := evaluator.Eval(context.Background())
			require.Error(t, err)
			require.Nil(t, response)
			require.Contains(t, err.Error(), "content is nil")
		})

		// Test script with execution error
		t.Run("script execution error", func(t *testing.T) {
			// Create a script that will intentionally cause an error
			scriptContent := `
def invalid_func():
    # This will cause a runtime error
    fail("intentional error")

invalid_func()
`
			_, evaluator := evalBuilder(t, scriptContent)
			response, err := evaluator.Eval(context.Background())
			require.Error(t, err)
			require.Nil(t, response)
			require.Contains(t, err.Error(), "intentional error")
		})
	})

	t.Run("metadata tests", func(t *testing.T) {
		// Test String() representation
		t.Run("String method", func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			evaluator := New(handler, nil)
			require.Equal(t, "starlark.Evaluator", evaluator.String())
		})
	})
}

// TestEvaluator_PrepareContext tests the PrepareContext method with various scenarios
func TestEvaluator_PrepareContext(t *testing.T) {
	t.Parallel()

	// Test cases
	tests := []struct {
		name         string
		setupExe     func(t *testing.T) *script.ExecutableUnit
		inputs       []any
		wantError    bool
		errorMessage string
	}{
		{
			name: "with successful provider",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()

				mockProvider := &MockProvider{}
				enrichedCtx := context.WithValue(
					context.Background(),
					constants.EvalData,
					"enriched",
				)
				mockProvider.On("AddDataToContext", mock.Anything, mock.Anything).
					Return(enrichedCtx, nil)

				return &script.ExecutableUnit{DataProvider: mockProvider}
			},
			inputs:    []any{map[string]any{"test": "data"}},
			wantError: false,
		},
		{
			name: "with provider error",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()

				mockProvider := &MockProvider{}
				expectedErr := fmt.Errorf("provider error")
				mockProvider.On("AddDataToContext", mock.Anything, mock.Anything).
					Return(nil, expectedErr)

				return &script.ExecutableUnit{DataProvider: mockProvider}
			},
			inputs:       []any{map[string]any{"test": "data"}},
			wantError:    true,
			errorMessage: "provider error",
		},
		{
			name: "nil provider",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()
				return &script.ExecutableUnit{DataProvider: nil}
			},
			inputs:       []any{map[string]any{"test": "data"}},
			wantError:    true,
			errorMessage: "no data provider available",
		},
		{
			name: "nil executable unit",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()
				return nil
			},
			inputs:       []any{map[string]any{"test": "data"}},
			wantError:    true,
			errorMessage: "no data provider available",
		},
	}

	// Run the test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stderr, nil)
			exe := tt.setupExe(t)

			evaluator := New(handler, exe)

			ctx := context.Background()
			result, err := evaluator.PrepareContext(ctx, tt.inputs...)

			if tt.wantError {
				require.Error(t, err)
				if tt.errorMessage != "" {
					require.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}

			// If using mocks, verify expectations
			if exe != nil && exe.DataProvider != nil {
				if mockProvider, ok := exe.DataProvider.(*MockProvider); ok {
					mockProvider.AssertExpectations(t)
				}
			}
		})
	}
}
