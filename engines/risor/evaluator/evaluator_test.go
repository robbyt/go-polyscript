package evaluator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/robbyt/go-polyscript/engines/risor/compiler"
	"github.com/robbyt/go-polyscript/engines/types"
	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script"
	"github.com/robbyt/go-polyscript/platform/script/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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
	data map[string]any,
) (context.Context, error) {
	args := m.Called(ctx, data)
	if ctx, ok := args.Get(0).(context.Context); ok {
		return ctx, args.Error(1)
	}
	return ctx, args.Error(1)
}

// MockLoader creates a simple loader for testing
type MockLoader struct {
	Content string
}

func (m *MockLoader) GetReader() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(m.Content)), nil
}

func (m *MockLoader) GetSourceURL() string {
	return "mock://source"
}

// MockContent implements ExecutableContent interface
type MockContent struct {
	Source  string
	Content any
}

func (m *MockContent) GetSource() string {
	return m.Source
}

func (m *MockContent) GetByteCode() any {
	return m.Content
}

func (m *MockContent) EngineType() types.Type {
	return types.Risor
}

// Helper function to create a test executable unit
func createTestExecutable(
	ctx context.Context,
	handler slog.Handler,
	ld loader.Loader,
	globals []string,
	provider data.Provider,
) (*script.ExecutableUnit, error) {
	c, err := compiler.New(
		compiler.WithLogHandler(handler),
		compiler.WithGlobals(globals),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create compiler: %w", err)
	}
	return script.NewExecutableUnit(ctx, handler, "test-id", ld, c, provider)
}

// TestEvaluator_Evaluate tests evaluating Risor scripts
func TestEvaluator_Evaluate(t *testing.T) {
	t.Parallel()

	// Define a test script that handles HTTP requests
	testScript := `
	let handle = function(request) {
		if (request == nil) {
			return error("request is nil")
		}
		if (request["Method"] == "POST") {
			return "post"
		}
		if (request["URL_Path"] == "/hello") {
			return true
		}
		return false
	}
	handle(ctx["request"])
	`

	t.Run("success cases", func(t *testing.T) {
		tests := []struct {
			name           string
			script         string
			requestMethod  string
			urlPath        string
			expectedType   data.Types
			expectedResult string
			expectedValue  any
		}{
			{
				name:           "GET request to /hello",
				script:         testScript,
				requestMethod:  "GET",
				urlPath:        "/hello",
				expectedType:   data.Types("bool"),
				expectedResult: "true",
				expectedValue:  true,
			},
			{
				name:           "POST request",
				script:         testScript,
				requestMethod:  "POST",
				urlPath:        "/hello",
				expectedType:   data.Types("string"),
				expectedResult: "\"post\"",
				expectedValue:  "post",
			},
			{
				name:           "GET request to unknown path",
				script:         testScript,
				requestMethod:  "GET",
				urlPath:        "/unknown",
				expectedType:   data.Types("bool"),
				expectedResult: "false",
				expectedValue:  false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
					Level: slog.LevelDebug,
				})

				// Create the loader and provider
				ld, err := loader.NewFromString(tt.script)
				require.NoError(t, err)
				ctxProvider := data.NewContextProvider(constants.EvalData)

				// Create executable unit and evaluator
				exe, err := createTestExecutable(t.Context(), handler, ld, []string{constants.Ctx}, ctxProvider)
				require.NoError(t, err)
				evaluator := New(handler, exe)
				require.NotNil(t, evaluator)

				// Create the request data
				req := httptest.NewRequest(tt.requestMethod, tt.urlPath, nil)
				rMap, err := helpers.RequestToMap(req)
				require.NoError(t, err)
				require.NotNil(t, rMap)

				// Create the context with eval data
				evalData := map[string]any{
					"request": rMap,
				}
				ctx := context.WithValue(t.Context(), constants.EvalData, evalData)

				// Execute the script
				response, err := evaluator.Eval(ctx)
				require.NoError(t, err)
				require.NotNil(t, response)

				// Verify the results
				require.Equal(t, tt.expectedType, response.Type())
				require.Equal(t, tt.expectedResult, response.Inspect())

				// Type-specific verification
				switch actualValue := response.Interface().(type) {
				case bool:
					expected, ok := tt.expectedValue.(bool)
					require.True(t, ok)
					require.Equal(t, expected, actualValue)
				case string:
					expected, ok := tt.expectedValue.(string)
					require.True(t, ok)
					require.Equal(t, expected, actualValue)
				default:
					require.Equal(t, tt.expectedValue, actualValue)
				}
			})
		}
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name         string
			setupExe     func() *script.ExecutableUnit
			errorMessage string
		}{
			{
				name: "nil executable unit",
				setupExe: func() *script.ExecutableUnit {
					return nil
				},
				errorMessage: "executable unit is nil",
			},
			{
				name: "nil bytecode",
				setupExe: func() *script.ExecutableUnit {
					return newExe(t, "test-id", &MockContent{Content: nil}, nil)
				},
				errorMessage: "bytecode is nil",
			},
			{
				name: "wrong bytecode type",
				setupExe: func() *script.ExecutableUnit {
					return newExe(t, "test-id", &MockContent{Content: "not a risor bytecode"}, nil)
				},
				errorMessage: "unable to type assert bytecode",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				handler := slog.NewTextHandler(os.Stderr, nil)
				exe := tt.setupExe()

				evaluator := &Evaluator{
					ctxKey:     constants.Ctx,
					execUnit:   exe,
					logHandler: handler,
					logger:     slog.New(handler),
				}

				ctx := t.Context()
				result, err := evaluator.Eval(ctx)

				require.Error(t, err)
				require.Nil(t, result)
				require.Contains(t, err.Error(), tt.errorMessage)
			})
		}
	})

	t.Run("load input data tests", func(t *testing.T) {
		tests := []struct {
			name         string
			setupExe     func() *script.ExecutableUnit
			setupCtx     func() context.Context
			expectError  bool
			errorMessage string
			expectEmpty  bool
		}{
			{
				name: "nil provider",
				setupExe: func() *script.ExecutableUnit {
					return nil
				},
				setupCtx: func() context.Context {
					return t.Context()
				},
				expectError: false,
				expectEmpty: true,
			},
			{
				name: "with provider error",
				setupExe: func() *script.ExecutableUnit {
					mockProvider := &MockProvider{}
					expectedErr := fmt.Errorf("provider error")
					mockProvider.On("GetData", mock.Anything).Return(nil, expectedErr)

					return newExe(t, "provider-error", nil, mockProvider)
				},
				setupCtx: func() context.Context {
					return t.Context()
				},
				expectError:  true,
				errorMessage: "provider error",
				expectEmpty:  true,
			},
			{
				name: "with empty data",
				setupExe: func() *script.ExecutableUnit {
					mockProvider := &MockProvider{}
					emptyData := map[string]any{}
					mockProvider.On("GetData", mock.Anything).Return(emptyData, nil)

					return newExe(t, "empty-data", nil, mockProvider)
				},
				setupCtx: func() context.Context {
					return t.Context()
				},
				expectError: false,
				expectEmpty: true,
			},
			{
				name: "with valid data",
				setupExe: func() *script.ExecutableUnit {
					mockProvider := &MockProvider{}
					validData := map[string]any{"test": "data"}
					mockProvider.On("GetData", mock.Anything).Return(validData, nil)

					return newExe(t, "valid-data", nil, mockProvider)
				},
				setupCtx: func() context.Context {
					return t.Context()
				},
				expectError: false,
				expectEmpty: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				handler := slog.NewTextHandler(os.Stderr, nil)
				exe := tt.setupExe()
				ctx := tt.setupCtx()

				evaluator := &Evaluator{
					ctxKey:     constants.Ctx,
					execUnit:   exe,
					logHandler: handler,
					logger:     slog.New(handler),
				}

				data, err := evaluator.loadInputData(ctx)

				if tt.expectError {
					require.Error(t, err)
					if tt.errorMessage != "" {
						require.Contains(t, err.Error(), tt.errorMessage)
					}
					require.Nil(t, data)
				} else {
					require.NoError(t, err)
					if tt.expectEmpty {
						assert.Empty(t, data)
					} else {
						assert.NotEmpty(t, data)
					}
				}

				// Verify mock expectations if we have a mockProvider
				if exe != nil && exe.GetDataProvider() != nil {
					if mockProvider, ok := exe.GetDataProvider().(*MockProvider); ok {
						mockProvider.AssertExpectations(t)
					}
				}
			})
		}
	})

	t.Run("metadata tests", func(t *testing.T) {
		// Test String method
		t.Run("String method", func(t *testing.T) {
			evaluator := &Evaluator{}
			require.Equal(t, "risor.Evaluator", evaluator.String())
		})

		// Test constructor with various options
		t.Run("constructor options", func(t *testing.T) {
			tests := []struct {
				name        string
				handler     slog.Handler
				checkLogger bool
			}{
				{
					name:        "with handler",
					handler:     slog.NewTextHandler(os.Stderr, nil),
					checkLogger: true,
				},
				{
					name:        "with nil handler",
					handler:     nil,
					checkLogger: false,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					exe := newExe(t, "new-test", nil, nil)
					evaluator := New(tt.handler, exe)

					require.NotNil(t, evaluator)
					require.Equal(t, constants.Ctx, evaluator.ctxKey)
					require.NotNil(t, evaluator.logger)
					require.NotNil(t, evaluator.logHandler)

					if tt.checkLogger && tt.handler != nil {
						require.Equal(t, tt.handler, evaluator.logHandler)
					}
				})
			}
		})
	})
}

func TestEvaluator_AddDataToContext(t *testing.T) {
	t.Parallel()

	// The test cases
	tests := []struct {
		name         string
		setupExe     func(t *testing.T) *script.ExecutableUnit
		input        map[string]any
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

				return newExe(t, "with-provider", nil, mockProvider)
			},
			input:     map[string]any{"test": "data"},
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

				return newExe(t, "with-provider-error", nil, mockProvider)
			},
			input:        map[string]any{"test": "data"},
			wantError:    true,
			errorMessage: "provider error",
		},
		{
			name: "nil provider",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()
				return newExe(t, "nil-provider", nil, nil)
			},
			input:        map[string]any{"test": "data"},
			wantError:    true,
			errorMessage: "no data provider available",
		},
		{
			name: "nil executable unit",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()
				return nil
			},
			input:        map[string]any{"test": "data"},
			wantError:    true,
			errorMessage: "no data provider available",
		},
	}

	// Run the test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stderr, nil)
			exe := tt.setupExe(t)

			evaluator := &Evaluator{
				ctxKey:     constants.Ctx,
				execUnit:   exe,
				logHandler: handler,
				logger:     slog.New(handler),
			}

			ctx := t.Context()
			result, err := evaluator.AddDataToContext(ctx, tt.input)

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
			if exe != nil && exe.GetDataProvider() != nil {
				if mockProvider, ok := exe.GetDataProvider().(*MockProvider); ok {
					mockProvider.AssertExpectations(t)
				}
			}
		})
	}
}

// TestEval_CancellationHaltsExecution verifies that cancelling the context
// passed to Eval() halts the running script within a bounded time. The
// script body is an unboundedly-long loop; only cancellation can return
// it within the test deadline (issue #125).
func TestEval_CancellationHaltsExecution(t *testing.T) {
	t.Parallel()

	handler := slog.NewTextHandler(io.Discard, nil)

	// Condition-only Risor for-loop with a counter that would take far
	// longer than the 2s test deadline to exhaust naturally; only ctx
	// cancellation lets Eval return early.
	// Risor v2 has no while/for-condition statements; iterate via a
	// lazy range and the .each higher-order. The VM checks ctx.Done()
	// periodically during execution (vm.go DefaultContextCheckInterval),
	// so cancellation halts inside the .each call. Range count chosen
	// so natural completion would far outrun the 2s test deadline.
	const script = `
range(1000000000).each(x => x)
"done"
`

	ld, err := loader.NewFromString(script)
	require.NoError(t, err)
	ctxProvider := data.NewContextProvider(constants.EvalData)
	exe, err := createTestExecutable(t.Context(), handler, ld, []string{constants.Ctx}, ctxProvider)
	require.NoError(t, err)
	eval := New(handler, exe)
	require.NotNil(t, eval)

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

// TestEval_ErrorTypeExposesRisorDetails verifies that errors returned from
// Eval expose the script-side diagnostic (Inspect() output of the error/function
// object) via errors.As and via the GetErrorDetails helper, and that the
// three-state (result, err) shape has been retired in favor of (nil, err).
func TestEval_ErrorTypeExposesRisorDetails(t *testing.T) {
	t.Parallel()

	buildEval := func(t *testing.T, script string) *Evaluator {
		t.Helper()
		handler := slog.NewTextHandler(io.Discard, nil)
		ld, err := loader.NewFromString(script)
		require.NoError(t, err)
		ctxProvider := data.NewContextProvider(constants.EvalData)
		exe, err := createTestExecutable(t.Context(), handler, ld, []string{constants.Ctx}, ctxProvider)
		require.NoError(t, err)
		eval := New(handler, exe)
		require.NotNil(t, eval)
		return eval
	}

	t.Run("script returns error surfaces diagnostic", func(t *testing.T) {
		t.Parallel()

		eval := buildEval(t, `error("user reason")`)
		ctx := context.WithValue(t.Context(), constants.EvalData, map[string]any{})

		result, err := eval.Eval(ctx)
		require.Error(t, err)
		require.Nil(t, result, "result should be nil — three-state shape retired")

		var rErr *Error
		require.ErrorAs(t, err, &rErr)
		require.Contains(t, rErr.ScriptResult, "user reason")
		require.Contains(t, GetErrorDetails(err), "user reason")
	})

	t.Run("script returns function surfaces diagnostic", func(t *testing.T) {
		t.Parallel()

		eval := buildEval(t, `x => x + 1`)
		ctx := context.WithValue(t.Context(), constants.EvalData, map[string]any{})

		result, err := eval.Eval(ctx)
		require.Error(t, err)
		require.Nil(t, result)

		var rErr *Error
		require.ErrorAs(t, err, &rErr)
		require.NotEmpty(t, rErr.ScriptResult)
		require.Contains(t, err.Error(), "function object returned from script")
	})

	t.Run("further wrapping preserves recovery", func(t *testing.T) {
		t.Parallel()

		eval := buildEval(t, `error("nested")`)
		ctx := context.WithValue(t.Context(), constants.EvalData, map[string]any{})

		_, err := eval.Eval(ctx)
		require.Error(t, err)

		wrapped := fmt.Errorf("upstream: %w", err)

		var rErr *Error
		require.ErrorAs(t, wrapped, &rErr)
		require.Contains(t, rErr.ScriptResult, "nested")
		require.Contains(t, GetErrorDetails(wrapped), "nested")
	})

	t.Run("VM exec failure populates Cause, leaves ScriptResult empty", func(t *testing.T) {
		t.Parallel()

		// Already-cancelled context drives the VM through be.exec and out
		// the err != nil branch deterministically — no reliance on
		// runtime-error script shapes.
		eval := buildEval(t, `range(1000000).each(x => x)`)
		ctx, cancel := context.WithCancel(t.Context())
		ctx = context.WithValue(ctx, constants.EvalData, map[string]any{})
		cancel()

		result, err := eval.Eval(ctx)
		require.Error(t, err)
		require.Nil(t, result)

		var rErr *Error
		require.ErrorAs(t, err, &rErr)
		require.Contains(t, rErr.Msg, "exec error")
		require.Empty(t, rErr.ScriptResult, "VM exec failure should not carry ScriptResult")
		require.Error(t, rErr.Cause, "Cause should expose the underlying VM error via Unwrap")
		// Cause should be unwrappable to the original ctx cancellation.
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("non-Risor failure leaves ScriptResult empty", func(t *testing.T) {
		t.Parallel()

		// Hitting the "executable unit is nil" path returns a *fmt.Errorf
		// without ever calling into the Risor VM — GetErrorDetails should
		// return "" and ErrorAs against *Error should not match.
		handler := slog.NewTextHandler(os.Stdout, nil)
		eval := &Evaluator{
			ctxKey:     constants.Ctx,
			execUnit:   nil,
			logHandler: handler,
			logger:     slog.New(handler),
		}

		_, err := eval.Eval(t.Context())
		require.Error(t, err)

		require.Empty(t, GetErrorDetails(err))

		var rErr *Error
		require.NotErrorAs(t, err, &rErr)
	})
}
