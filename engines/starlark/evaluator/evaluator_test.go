package evaluator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/synctest"
	"time"

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

func (m *MockProvider) AddDataToContext(
	ctx context.Context,
	data ...map[string]any,
) (context.Context, error) {
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
					"request": rMap,
				}

				ctx := context.WithValue(t.Context(), constants.EvalData, evalData)

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

			response, err := evaluator.Eval(t.Context())
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

			response, err := evaluator.Eval(t.Context())
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
			response, err := evaluator.Eval(t.Context())
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

// TestEval_NoGoroutineLeak verifies that Eval() with a non-cancellable context does not leak
// goroutines. synctest.Test only returns once every goroutine spawned inside the bubble has
// exited; a stray goroutine blocked on <-ctx.Done() (the scenario PR #81 fixed by switching to
// context.AfterFunc) would be reported as a leak with a clear diagnostic.
func TestEval_NoGoroutineLeak(t *testing.T) {
	t.Parallel()

	scriptContent := `_ = 1 + 1`
	_, evaluator := evalBuilder(t, scriptContent)

	synctest.Test(t, func(t *testing.T) {
		// Use context.Background() so ctx.Done() is nil — this is the exact scenario where a
		// bare goroutine waiting on <-ctx.Done() would block forever and leak. t.Context()
		// would defeat the test by giving a cancellable context.
		for range 100 {
			//nolint:usetesting // intentional — see comment above
			ctx := context.WithValue(context.Background(), constants.EvalData, map[string]any{})
			_, err := evaluator.Eval(ctx)
			require.NoError(t, err)
		}
	})
}

// TestEvaluator_AddDataToContext tests the AddDataToContext method with various scenarios
func TestEvaluator_AddDataToContext(t *testing.T) {
	t.Parallel()

	// Test cases
	tests := []struct {
		name         string
		setupExe     func(t *testing.T) *script.ExecutableUnit
		inputs       []map[string]any
		wantError    bool
		errorMessage string
	}{
		{
			name: "with successful provider",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()

				mockProvider := &MockProvider{}
				enrichedCtx := context.WithValue(
					t.Context(),
					constants.EvalData,
					"enriched",
				)
				mockProvider.On("AddDataToContext", mock.Anything, mock.Anything).
					Return(enrichedCtx, nil)

				return &script.ExecutableUnit{DataProvider: mockProvider}
			},
			inputs:    []map[string]any{{"test": "data"}},
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
			inputs:       []map[string]any{{"test": "data"}},
			wantError:    true,
			errorMessage: "provider error",
		},
		{
			name: "nil provider",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()
				return &script.ExecutableUnit{DataProvider: nil}
			},
			inputs:       []map[string]any{{"test": "data"}},
			wantError:    true,
			errorMessage: "no data provider available",
		},
		{
			name: "nil executable unit",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()
				return nil
			},
			inputs:       []map[string]any{{"test": "data"}},
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

			ctx := t.Context()
			result, err := evaluator.AddDataToContext(ctx, tt.inputs...)

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

// TestEval_CancellationHaltsExecution is the regression companion to PR #81
// (issue #125). PR #81 added a context.AfterFunc-registered thread.Cancel
// in engines/starlark/evaluator/exec(); this test confirms the registered
// cancellation actually halts a running script within a bounded time. The
// script body is an unboundedly-long loop; only cancellation can return
// it within the test deadline.
func TestEval_CancellationHaltsExecution(t *testing.T) {
	t.Parallel()

	// Lazy range(); cancellation halts at the next instruction boundary
	// via the AfterFunc-registered thread.Cancel().
	const scriptContent = `
def spin():
    for i in range(1000000000000):
        pass
    return "done"

result = spin()
`
	_, eval := evalBuilder(t, scriptContent)

	// evalBuilder uses a ContextProvider with constants.EvalData; populate
	// it with an empty map so loadInputData has the value to read.
	ctx, cancel := context.WithCancel(t.Context())
	ctx = context.WithValue(ctx, constants.EvalData, map[string]any{})

	done := make(chan error, 1)
	go func() {
		_, err := eval.Eval(ctx)
		done <- err
	}()

	// Give the engine a moment to enter the spin loop, then cancel.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.Error(t, err)
		require.True(t,
			errors.Is(err, context.Canceled) ||
				strings.Contains(err.Error(), "context canceled") ||
				strings.Contains(err.Error(), "cancel"),
			"expected cancellation-shaped error, got: %v", err,
		)
	case <-time.After(2 * time.Second):
		t.Fatal("Eval did not return within 2s after cancel; cancellation unresponsive")
	}
}

// TestEval_ErrorTypeExposesStarlarkDetails verifies that errors returned from
// Eval expose the underlying *starlark.EvalError via errors.As and via the
// GetErrorDetails helper, including through additional fmt.Errorf wrapping.
func TestEval_ErrorTypeExposesStarlarkDetails(t *testing.T) {
	t.Parallel()

	t.Run("fail() in script surfaces EvalError", func(t *testing.T) {
		t.Parallel()

		const scriptContent = `fail("user reason")`
		_, eval := evalBuilder(t, scriptContent)

		_, err := eval.Eval(t.Context())
		require.Error(t, err)

		var evalErrWrap *Error
		require.ErrorAs(t, err, &evalErrWrap)
		require.NotNil(t, evalErrWrap.EvalErr, "EvalErr should be populated for fail()")
		require.Contains(t, evalErrWrap.EvalErr.Msg, "user reason")
		require.NotEmpty(t, evalErrWrap.EvalErr.Backtrace(), "Backtrace should be non-empty")

		details := GetErrorDetails(err)
		require.NotNil(t, details)
		require.Same(t, evalErrWrap.EvalErr, details, "GetErrorDetails returns the same *starlark.EvalError")
	})

	t.Run("further wrapping preserves recovery", func(t *testing.T) {
		t.Parallel()

		const scriptContent = `fail("nested")`
		_, eval := evalBuilder(t, scriptContent)

		_, err := eval.Eval(t.Context())
		require.Error(t, err)

		wrapped := fmt.Errorf("upstream: %w", err)

		var evalErrWrap *Error
		require.ErrorAs(t, wrapped, &evalErrWrap)
		require.NotNil(t, evalErrWrap.EvalErr)
		require.Contains(t, evalErrWrap.EvalErr.Msg, "nested")

		require.NotNil(t, GetErrorDetails(wrapped))
	})

	t.Run("auto-call failure surfaces EvalError", func(t *testing.T) {
		t.Parallel()

		// Script's last value is a callable; auto-call invokes it; the call
		// raises via fail() and the evaluator wraps it as *Error.
		const scriptContent = `
def boom():
    fail("from callable")

_ = boom
`
		_, eval := evalBuilder(t, scriptContent)

		_, err := eval.Eval(t.Context())
		require.Error(t, err)
		require.Contains(t, err.Error(), "error calling function")

		details := GetErrorDetails(err)
		require.NotNil(t, details)
		require.Contains(t, details.Msg, "from callable")
	})

	t.Run("non-Starlark failure leaves EvalErr nil", func(t *testing.T) {
		t.Parallel()

		// Hitting the "executable unit is nil" path returns a *fmt.Errorf
		// without ever calling into Starlark — GetErrorDetails should return
		// nil and a bare err != nil check still works.
		handler := slog.NewTextHandler(os.Stdout, nil)
		eval := New(handler, nil)

		_, err := eval.Eval(t.Context())
		require.Error(t, err)

		require.Nil(t, GetErrorDetails(err))

		var evalErrWrap *Error
		require.NotErrorAs(t, err, &evalErrWrap)
	})
}
