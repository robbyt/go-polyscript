package evaluator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/engines/extism/adapters"
	"github.com/robbyt/go-polyscript/engines/extism/compiler"
	"github.com/robbyt/go-polyscript/engines/extism/internal"
	engineTypes "github.com/robbyt/go-polyscript/engines/types"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockCompiledPlugin is a mock implementation of adapters.CompiledPlugin
type MockCompiledPlugin struct {
	mock.Mock
}

func (m *MockCompiledPlugin) Instance(
	ctx context.Context,
	cfg extismSDK.PluginInstanceConfig,
) (adapters.PluginInstance, error) {
	args := m.Called(ctx, cfg)
	return args.Get(0).(adapters.PluginInstance), args.Error(1)
}

func (m *MockCompiledPlugin) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// createMockExecutable creates a real compiler.Executable with our mock plugin
func createMockExecutable(
	mockPlugin adapters.CompiledPlugin,
	entryPoint string,
) *compiler.Executable {
	// Create some mock WASM bytes
	wasmBytes := []byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00}

	// Use the real Executable type with our mock plugin
	return compiler.NewExecutable(wasmBytes, mockPlugin, entryPoint)
}

// mockErrProvider implements the data.Provider interface and always returns an error
type mockErrProvider struct {
	err error
}

func (m *mockErrProvider) GetData(ctx context.Context) (map[string]any, error) {
	return nil, m.err
}

func (m *mockErrProvider) AddDataToContext(
	ctx context.Context,
	data ...map[string]any,
) (context.Context, error) {
	return ctx, m.err
}

// mockMapProvider returns a fixed data map without erroring. Tests use this
// to control the bytes that flow into internal.ConvertToExtismFormat.
type mockMapProvider struct {
	data map[string]any
}

func (m *mockMapProvider) GetData(ctx context.Context) (map[string]any, error) {
	return m.data, nil
}

func (m *mockMapProvider) AddDataToContext(
	ctx context.Context,
	data ...map[string]any,
) (context.Context, error) {
	return ctx, nil
}

// mockPluginInstance is a mock implementation of the adapters.PluginInstance interface
type mockPluginInstance struct {
	exitCode   uint32
	output     []byte
	callErr    error
	closeErr   error
	wasCalled  bool
	wasClosed  bool
	cancelFunc func()
}

func (m *mockPluginInstance) CallWithContext(
	ctx context.Context,
	functionName string,
	input []byte,
) (uint32, []byte, error) {
	m.wasCalled = true
	// Execute the cancel function if provided (to simulate context cancellation)
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	// Check if the context was canceled
	if ctx.Err() != nil {
		return 0, nil, ctx.Err()
	}
	return m.exitCode, m.output, m.callErr
}

func (m *mockPluginInstance) Call(name string, data []byte) (uint32, []byte, error) {
	m.wasCalled = true
	return m.exitCode, m.output, m.callErr
}

func (m *mockPluginInstance) FunctionExists(name string) bool {
	return true
}

func (m *mockPluginInstance) Close(ctx context.Context) error {
	m.wasClosed = true
	return m.closeErr
}

type mockExecutableContent struct {
	engineType engineTypes.Type
	source     string
	bytecode   any
}

func (m *mockExecutableContent) EngineType() engineTypes.Type {
	return m.engineType
}

func (m *mockExecutableContent) GetSource() string {
	return m.source
}

func (m *mockExecutableContent) GetByteCode() any {
	return m.bytecode
}

// TestEvaluator_Evaluate tests evaluating WASM scripts with Extism
func TestEvaluator_Evaluate(t *testing.T) {
	t.Parallel()

	t.Run("success cases", func(t *testing.T) {
		// Test successful JSON response
		t.Run("successful execution with JSON output", func(t *testing.T) {
			// Skip this test in CI environments that may not support WASM
			if os.Getenv("CI") != "" {
				t.Skip("Skipping WASM test in CI environment")
			}

			handler := slog.NewTextHandler(os.Stdout, nil)

			// Create context provider
			ctxProvider := data.NewContextProvider(constants.EvalData)

			// Create mock plugin
			mockPlugin := new(MockCompiledPlugin)
			mockInstance := &mockPluginInstance{
				exitCode: 0, // Success
				output:   []byte(`{"result":"success", "value": 42}`),
			}
			mockPlugin.On("Instance", mock.Anything, mock.Anything).Return(mockInstance, nil)
			mockPlugin.On("Close", mock.Anything).Return(nil)

			// Create a real compiler.Executable with our mock plugin
			content := createMockExecutable(mockPlugin, "main")

			// Create a mock executable
			exe := &script.ExecutableUnit{
				ID:           "test-json-success",
				DataProvider: ctxProvider,
				Content:      content,
			}

			evaluator := New(handler, exe)

			ctx := t.Context()
			evalData := map[string]any{"test": "data"}
			ctx = context.WithValue(ctx, constants.EvalData, evalData)

			response, err := evaluator.Eval(ctx)
			require.NoError(t, err)
			require.NotNil(t, response)

			// Verify the response
			resultMap, ok := response.Interface().(map[string]any)
			require.True(t, ok, "Expected map response")
			require.Contains(t, resultMap, "result")
			require.Equal(t, "success", resultMap["result"])
			require.Contains(t, resultMap, "value")
			require.InDelta(t, float64(42), resultMap["value"], 0.0001)
		})

		// Test successful string response
		t.Run("successful execution with string output", func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			ctxProvider := data.NewContextProvider(constants.EvalData)

			mockPlugin := new(MockCompiledPlugin)
			mockInstance := &mockPluginInstance{
				exitCode: 0,
				output:   []byte(`Hello, World!`), // Plain text
			}
			mockPlugin.On("Instance", mock.Anything, mock.Anything).Return(mockInstance, nil)
			mockPlugin.On("Close", mock.Anything).Return(nil)

			content := createMockExecutable(mockPlugin, "main")
			exe := &script.ExecutableUnit{
				ID:           "test-string-success",
				DataProvider: ctxProvider,
				Content:      content,
			}

			evaluator := New(handler, exe)
			ctx := t.Context()
			evalData := map[string]any{"test": "data"}
			ctx = context.WithValue(ctx, constants.EvalData, evalData)

			response, err := evaluator.Eval(ctx)
			require.NoError(t, err)
			require.NotNil(t, response)

			// Verify the string response
			require.Equal(t, "Hello, World!", response.Interface())
		})

		// Test load input data with various context values
		t.Run("load input data", func(t *testing.T) {
			tests := []struct {
				name          string
				ctxData       any
				expectedEmpty bool
			}{
				{
					name:          "empty context",
					ctxData:       nil,
					expectedEmpty: true,
				},
				{
					name: "valid data",
					ctxData: map[string]any{
						"foo": "bar",
						"nested": map[string]any{
							"a": 1,
							"b": 2,
						},
					},
					expectedEmpty: false,
				},
				{
					name:          "empty data",
					ctxData:       map[string]any{},
					expectedEmpty: true,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					handler := slog.NewTextHandler(os.Stdout, nil)
					ctxProvider := data.NewContextProvider(constants.EvalData)
					dummyExe := &script.ExecutableUnit{
						DataProvider: ctxProvider,
					}

					evaluator := New(handler, dummyExe)
					ctx := t.Context()

					if tt.ctxData != nil {
						ctx = context.WithValue(ctx, constants.EvalData, tt.ctxData)
					}

					// Test the loadInputData method
					result, err := evaluator.loadInputData(ctx)
					require.NoError(t, err)

					if tt.expectedEmpty {
						assert.Empty(t, result)
					} else {
						assert.NotEmpty(t, result)
						if validMap, ok := tt.ctxData.(map[string]any); ok {
							assert.Equal(t, validMap, result)
						}
					}
				})
			}
		})

		// Test how input data is formatted for Extism
		t.Run("input data formatting", func(t *testing.T) {
			// Create a test map that simulates data from our providers
			inputData := map[string]any{
				"initial": "top-level-value", // Static data at top level
				"input_data": map[string]any{ // Dynamic data nested under input_data
					"input":   "API User",
					"request": map[string]any{}, // HTTP request data nested under input_data
				},
			}

			// Convert the input data for Extism
			jsonBytes, err := internal.ConvertToExtismFormat(inputData)
			require.NoError(t, err)
			require.NotNil(t, jsonBytes)

			// Verify current behavior
			expected := `{"initial":"top-level-value","input_data":{"input":"API User","request":{}}}`
			assert.JSONEq(t, expected, string(jsonBytes))
		})
	})

	t.Run("error cases", func(t *testing.T) {
		// Test nil executable unit
		t.Run("nil executable unit", func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			evaluator := New(handler, nil)

			ctx := t.Context()
			_, err := evaluator.Eval(ctx)

			require.Error(t, err)
			require.Contains(t, err.Error(), "executable unit is nil")
		})

		// Test nil bytecode
		t.Run("nil bytecode", func(t *testing.T) {
			mockContent := &mockExecutableContent{
				engineType: engineTypes.Extism,
				source:     "invalid wasm",
				bytecode:   nil, // Nil bytecode will cause error
			}

			handler := slog.NewTextHandler(os.Stdout, nil)
			ctxProvider := data.NewContextProvider(constants.EvalData)

			exe := &script.ExecutableUnit{
				ID:           "test-case",
				Content:      mockContent,
				DataProvider: ctxProvider,
			}

			evaluator := New(handler, exe)

			ctx := t.Context()
			_, err := evaluator.Eval(ctx)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "bytecode is nil")
		})

		// Test invalid content type
		t.Run("invalid content type", func(t *testing.T) {
			mockContent := &mockExecutableContent{
				engineType: engineTypes.Extism,
				source:     "invalid wasm",
				bytecode:   []byte{0x00}, // Not a valid WASM plugin
			}

			handler := slog.NewTextHandler(os.Stdout, nil)
			ctxProvider := data.NewContextProvider(constants.EvalData)

			exe := &script.ExecutableUnit{
				ID:           "test-case",
				Content:      mockContent,
				DataProvider: ctxProvider,
			}

			evaluator := New(handler, exe)

			ctx := t.Context()
			_, err := evaluator.Eval(ctx)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid executable type")
		})

		// Test context cancellation
		t.Run("context cancellation", func(t *testing.T) {
			// Create a cancel context
			ctx, cancel := context.WithCancel(t.Context())

			// Create mock plugin that will check for cancellation
			mockPlugin := new(MockCompiledPlugin)
			mockInstance := &mockPluginInstance{
				cancelFunc: func() {
					// This will be called during execution to cancel the context
					cancel()
				},
				callErr: context.Canceled,
			}
			mockPlugin.On("Instance", mock.Anything, mock.Anything).Return(mockInstance, nil)
			mockPlugin.On("Close", mock.Anything).Return(nil)

			// Create a real compiler.Executable with our mock plugin
			content := createMockExecutable(mockPlugin, "main")

			// Create executor unit
			handler := slog.NewTextHandler(os.Stdout, nil)
			execUnit := &script.ExecutableUnit{
				ID:           "test-cancel",
				Content:      content,
				DataProvider: data.NewContextProvider(constants.EvalData),
			}

			evaluator := New(handler, execUnit)

			// Add test data to context
			ctx = context.WithValue(ctx, constants.EvalData, map[string]any{"test": "data"})

			// Call Eval, which should be cancelled during execution
			result, err := evaluator.Eval(ctx)

			require.Error(t, err, "Expected cancellation error")
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "execution")

			// Instance should have been called
			mockPlugin.AssertCalled(t, "Instance", mock.Anything, mock.Anything)

			// Instance should have been closed
			assert.True(t, mockInstance.wasClosed)
		})

		// Test execution with non-zero exit code
		t.Run("non-zero exit code", func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			ctxProvider := data.NewContextProvider(constants.EvalData)

			mockPlugin := new(MockCompiledPlugin)
			mockInstance := &mockPluginInstance{
				exitCode: 1, // Error exit code
				output:   []byte(`{"error":"something went wrong"}`),
			}
			mockPlugin.On("Instance", mock.Anything, mock.Anything).Return(mockInstance, nil)
			mockPlugin.On("Close", mock.Anything).Return(nil)

			content := createMockExecutable(mockPlugin, "main")
			exe := &script.ExecutableUnit{
				ID:           "test-error-exit",
				DataProvider: ctxProvider,
				Content:      content,
			}

			evaluator := New(handler, exe)
			ctx := t.Context()
			evalData := map[string]any{"test": "data"}
			ctx = context.WithValue(ctx, constants.EvalData, evalData)

			_, err := evaluator.Eval(ctx)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "non-zero exit code")
			// Output bytes should now be surfaced in the error so the host
			// can diagnose the misbehaving plugin (issue #94).
			assert.Contains(t, err.Error(), "something went wrong")
		})

		// Regression for issue #122: WithExitOutputMaxBytes flows from
		// Evaluator construction through exec → execHelper → formatExitOutput.
		// A negative cap disables truncation, so a multi-KiB plugin payload
		// makes it into the error string unchanged.
		t.Run("non-zero exit code with WithExitOutputMaxBytes(-1)", func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			ctxProvider := data.NewContextProvider(constants.EvalData)

			payload := bytes.Repeat([]byte("Y"), defaultExitOutputMaxBytes*4)

			mockPlugin := new(MockCompiledPlugin)
			mockInstance := &mockPluginInstance{
				exitCode: 7,
				output:   payload,
			}
			mockPlugin.On("Instance", mock.Anything, mock.Anything).Return(mockInstance, nil)
			mockPlugin.On("Close", mock.Anything).Return(nil)

			content := createMockExecutable(mockPlugin, "main")
			exe := &script.ExecutableUnit{
				ID:           "test-error-exit-uncapped",
				DataProvider: ctxProvider,
				Content:      content,
			}

			evaluator := New(handler, exe, WithExitOutputMaxBytes(-1))
			_, err := evaluator.Eval(t.Context())

			require.Error(t, err)
			assert.Contains(t, err.Error(), "non-zero exit code: 7")
			// No truncation tail.
			assert.NotContains(t, err.Error(), "truncated from")
			// The full payload survived end-to-end.
			assert.Contains(t, err.Error(), string(payload[:32]))
			assert.Contains(t, err.Error(), string(payload[len(payload)-32:]))
		})

		// Test error creating plugin instance
		t.Run("error creating plugin instance", func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			mockPlugin := new(MockCompiledPlugin)
			mockInstance := &mockPluginInstance{}
			mockPlugin.On("Instance", mock.Anything, mock.Anything).
				Return(mockInstance, errors.New("instance creation error"))
			mockPlugin.On("Close", mock.Anything).Return(nil)

			content := createMockExecutable(mockPlugin, "main")
			exe := &script.ExecutableUnit{
				ID:           "test-instance-error",
				DataProvider: data.NewContextProvider(constants.EvalData),
				Content:      content,
			}

			evaluator := New(handler, exe)
			ctx := t.Context()

			_, err := evaluator.Eval(ctx)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create plugin instance")
		})

		// Eval should wrap the data-provider's error when loadInputData fails,
		// without ever reaching plugin.Instance.
		t.Run("load input data error", func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			mockPlugin := new(MockCompiledPlugin)
			content := createMockExecutable(mockPlugin, "main")

			exe := &script.ExecutableUnit{
				ID:           "test-load-input-error",
				DataProvider: &mockErrProvider{err: errors.New("provider boom")},
				Content:      content,
			}

			evaluator := New(handler, exe)
			_, err := evaluator.Eval(t.Context())

			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to get input data")
			assert.Contains(t, err.Error(), "provider boom")
			mockPlugin.AssertNotCalled(t, "Instance", mock.Anything, mock.Anything)
		})

		// Eval should wrap json.Marshal errors from internal.ConvertToExtismFormat
		// without reaching plugin.Instance. A chan value is not JSON-marshalable.
		t.Run("convert to extism format error", func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			mockPlugin := new(MockCompiledPlugin)
			content := createMockExecutable(mockPlugin, "main")

			exe := &script.ExecutableUnit{
				ID: "test-convert-format-error",
				DataProvider: &mockMapProvider{data: map[string]any{
					"bad": make(chan int),
				}},
				Content: content,
			}

			evaluator := New(handler, exe)
			_, err := evaluator.Eval(t.Context())

			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to marshal input data")
			mockPlugin.AssertNotCalled(t, "Instance", mock.Anything, mock.Anything)
		})
	})

	t.Run("metadata tests", func(t *testing.T) {
		// Test nil handler fallback
		t.Run("nil handler fallback", func(t *testing.T) {
			// Create mock plugin
			mockPlugin := new(MockCompiledPlugin)
			mockPlugin.On("Close", mock.Anything).Return(nil)

			// Create a real compiler.Executable with our mock plugin
			content := createMockExecutable(mockPlugin, "main")

			exe := &script.ExecutableUnit{
				ID:           "test-nil-handler",
				DataProvider: data.NewContextProvider(constants.EvalData),
				Content:      content,
			}

			// Create with nil handler
			evaluator := New(nil, exe)

			// Shouldn't panic
			require.NotNil(t, evaluator)
			require.NotNil(t, evaluator.logger)
			require.NotNil(t, evaluator.logHandler)
		})

		// Test String method
		t.Run("String method", func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			evaluator := New(handler, nil)

			// Test the string representation
			strRep := evaluator.String()
			require.Equal(t, "extism.Evaluator", strRep)
		})

		// Test the exec helper function
		t.Run("exec helper", func(t *testing.T) {
			tests := []struct {
				name           string
				setup          func() (*mockPluginInstance, context.Context, context.CancelFunc)
				entryPoint     string
				input          []byte
				maxBytes       int // exit-output cap; zero uses defaultExitOutputMaxBytes
				wantErr        bool
				errContainsAll []string
				errExcludes    []string
			}{
				{
					name: "successful execution",
					setup: func() (*mockPluginInstance, context.Context, context.CancelFunc) {
						ctx, cancel := context.WithCancel(t.Context())
						return &mockPluginInstance{
							exitCode: 0,
							output:   []byte(`{"result": "success", "count": 42}`),
						}, ctx, cancel
					},
					entryPoint: "main",
					input:      []byte(`{"key":"value"}`),
					wantErr:    false,
				},
				{
					name: "non-zero exit code",
					setup: func() (*mockPluginInstance, context.Context, context.CancelFunc) {
						ctx, cancel := context.WithCancel(t.Context())
						return &mockPluginInstance{
							exitCode: 1,
							output:   []byte(`{"error": "something went wrong"}`),
						}, ctx, cancel
					},
					entryPoint:     "main",
					input:          []byte(`{"key":"value"}`),
					wantErr:        true,
					errContainsAll: []string{"non-zero exit code", "something went wrong"},
				},
				{
					name: "non-zero exit code with empty output",
					setup: func() (*mockPluginInstance, context.Context, context.CancelFunc) {
						ctx, cancel := context.WithCancel(t.Context())
						return &mockPluginInstance{
							exitCode: 1,
							output:   nil,
						}, ctx, cancel
					},
					entryPoint:     "main",
					input:          []byte(`{"key":"value"}`),
					wantErr:        true,
					errContainsAll: []string{"non-zero exit code: 1"},
					// Empty output must not produce a noisy '(output: "")' tail.
					errExcludes: []string{"(output:"},
				},
				{
					// Sized as a multiple of defaultExitOutputMaxBytes so the
					// truncation branch keeps firing if the default is raised.
					name: "non-zero exit code with truncated output (default cap)",
					setup: func() (*mockPluginInstance, context.Context, context.CancelFunc) {
						ctx, cancel := context.WithCancel(t.Context())
						return &mockPluginInstance{
							exitCode: 2,
							output:   bytes.Repeat([]byte("X"), defaultExitOutputMaxBytes*2),
						}, ctx, cancel
					},
					entryPoint: "main",
					input:      []byte(`{"key":"value"}`),
					wantErr:    true,
					errContainsAll: []string{
						"non-zero exit code: 2",
						fmt.Sprintf("truncated from %d bytes", defaultExitOutputMaxBytes*2),
					},
				},
				{
					// Negative cap disables truncation; the full output is
					// quoted into the error and there's no "truncated from"
					// tail.
					name: "non-zero exit code with disabled cap",
					setup: func() (*mockPluginInstance, context.Context, context.CancelFunc) {
						ctx, cancel := context.WithCancel(t.Context())
						return &mockPluginInstance{
							exitCode: 3,
							output:   bytes.Repeat([]byte("Y"), defaultExitOutputMaxBytes*4),
						}, ctx, cancel
					},
					entryPoint:     "main",
					input:          []byte(`{"key":"value"}`),
					maxBytes:       -1,
					wantErr:        true,
					errContainsAll: []string{"non-zero exit code: 3"},
					errExcludes:    []string{"truncated from"},
				},
				{
					// Small custom cap is honored and truncation kicks in
					// before reaching the default.
					name: "non-zero exit code with custom small cap",
					setup: func() (*mockPluginInstance, context.Context, context.CancelFunc) {
						ctx, cancel := context.WithCancel(t.Context())
						return &mockPluginInstance{
							exitCode: 4,
							output:   []byte("0123456789ABCDEFGHIJ"),
						}, ctx, cancel
					},
					entryPoint: "main",
					input:      []byte(`{"key":"value"}`),
					maxBytes:   4,
					wantErr:    true,
					errContainsAll: []string{
						"non-zero exit code: 4",
						"truncated from 20 bytes",
					},
				},
				{
					name: "execution error",
					setup: func() (*mockPluginInstance, context.Context, context.CancelFunc) {
						ctx, cancel := context.WithCancel(t.Context())
						return &mockPluginInstance{
							callErr: errors.New("execution failed"),
						}, ctx, cancel
					},
					entryPoint:     "main",
					input:          []byte(`{"key":"value"}`),
					wantErr:        true,
					errContainsAll: []string{"execution failed"},
				},
				{
					name: "context cancellation",
					setup: func() (*mockPluginInstance, context.Context, context.CancelFunc) {
						ctx, cancel := context.WithCancel(t.Context())
						mock := &mockPluginInstance{
							cancelFunc: cancel, // This will cancel the context during execution
							callErr:    context.Canceled,
						}
						return mock, ctx, cancel
					},
					entryPoint:     "main",
					input:          []byte(`{"key":"value"}`),
					wantErr:        true,
					errContainsAll: []string{"cancelled"},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					mockInstance, ctx, cancel := tt.setup()
					defer cancel()

					logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
					result, execTime, err := execHelper(
						ctx,
						logger,
						mockInstance,
						tt.entryPoint,
						tt.input,
						tt.maxBytes,
					)

					// Verify the mock was called
					assert.True(
						t,
						mockInstance.wasCalled,
						"Expected the mock instance to be called",
					)

					if tt.wantErr {
						require.Error(t, err)
						for _, s := range tt.errContainsAll {
							assert.Contains(t, err.Error(), s)
						}
						for _, s := range tt.errExcludes {
							assert.NotContains(t, err.Error(), s)
						}
					} else {
						require.NoError(t, err)
						assert.NotNil(t, result)
					}

					// Execution time should always be measured
					assert.Positive(t, execTime.Nanoseconds())
				})
			}
		})
	})
}

// TestEvaluator_AddDataToContext tests the AddDataToContext method with various scenarios
func TestEvaluator_AddDataToContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupExe    func(t *testing.T) *script.ExecutableUnit
		inputs      []map[string]any
		wantError   bool
		expectedErr string
	}{
		{
			name: "nil data provider",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()

				mockPlugin := new(MockCompiledPlugin)
				mockPlugin.On("Close", mock.Anything).Return(nil)
				content := createMockExecutable(mockPlugin, "main")

				return &script.ExecutableUnit{
					ID:           "test-nil-provider",
					DataProvider: nil,
					Content:      content,
				}
			},
			inputs:      []map[string]any{{"test": "data"}},
			wantError:   true,
			expectedErr: "no data provider available",
		},
		{
			name: "valid simple data",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()

				mockPlugin := new(MockCompiledPlugin)
				mockPlugin.On("Close", mock.Anything).Return(nil)
				content := createMockExecutable(mockPlugin, "main")

				return &script.ExecutableUnit{
					ID:           "test-valid-data",
					DataProvider: data.NewContextProvider(constants.EvalData),
					Content:      content,
				}
			},
			inputs:    []map[string]any{{"test": "data"}},
			wantError: false,
		},
		{
			name: "empty input",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()

				mockPlugin := new(MockCompiledPlugin)
				mockPlugin.On("Close", mock.Anything).Return(nil)
				content := createMockExecutable(mockPlugin, "main")

				return &script.ExecutableUnit{
					ID:           "test-empty-input",
					DataProvider: data.NewContextProvider(constants.EvalData),
					Content:      content,
				}
			},
			inputs:    []map[string]any{{}},
			wantError: false,
		},
		{
			name: "nil executable unit",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()
				return nil
			},
			inputs:      []map[string]any{{"test": "data"}},
			wantError:   true,
			expectedErr: "no data provider available",
		},
		{
			name: "with error throwing provider",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()

				mockPlugin := new(MockCompiledPlugin)
				mockPlugin.On("Close", mock.Anything).Return(nil)
				content := createMockExecutable(mockPlugin, "main")

				mockProvider := &mockErrProvider{
					err: errors.New("provider error"),
				}
				return &script.ExecutableUnit{
					ID:           "test-err-provider",
					DataProvider: mockProvider,
					Content:      content,
				}
			},
			inputs:      []map[string]any{{"test": "data"}},
			wantError:   true,
			expectedErr: "provider error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			exe := tt.setupExe(t)
			evaluator := New(handler, exe)

			ctx := t.Context()
			enrichedCtx, err := evaluator.AddDataToContext(ctx, tt.inputs...)

			// Check error expectations
			if tt.wantError {
				require.Error(t, err)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
			}

			// Even with errors, we should get a context back (might be the original)
			require.NotNil(t, enrichedCtx)
		})
	}
}

// TestFormatExitOutput covers the helper that builds the "(output: ...)"
// suffix appended to non-zero-exit error messages.
func TestFormatExitOutput(t *testing.T) {
	t.Parallel()

	t.Run("empty output produces no suffix regardless of cap", func(t *testing.T) {
		assert.Empty(t, formatExitOutput(nil, 0))
		assert.Empty(t, formatExitOutput([]byte{}, 0))
		assert.Empty(t, formatExitOutput(nil, 100))
		assert.Empty(t, formatExitOutput(nil, -1))
	})

	t.Run("short output is quoted in full with default cap", func(t *testing.T) {
		assert.Equal(t, ` (output: "boom")`, formatExitOutput([]byte("boom"), 0))
	})

	t.Run("output exactly at default cap is not truncated", func(t *testing.T) {
		payload := bytes.Repeat([]byte("a"), defaultExitOutputMaxBytes)
		got := formatExitOutput(payload, 0)
		assert.Contains(t, got, "(output:")
		assert.NotContains(t, got, "truncated")
	})

	t.Run("output over default cap reports original byte count", func(t *testing.T) {
		payload := bytes.Repeat([]byte("a"), defaultExitOutputMaxBytes+5)
		got := formatExitOutput(payload, 0)
		assert.Contains(t, got, "truncated from")
		assert.Contains(t, got, fmt.Sprintf("%d bytes", len(payload)))
	})

	t.Run("zero cap falls back to default", func(t *testing.T) {
		// A 5-byte payload sits well below the 1024-byte default cap, so
		// passing 0 must not truncate it.
		got := formatExitOutput([]byte("short"), 0)
		assert.Equal(t, ` (output: "short")`, got)
	})

	t.Run("negative cap disables truncation", func(t *testing.T) {
		// A payload several multiples of the default cap must come back
		// unchanged when the caller asks for no cap.
		payload := bytes.Repeat([]byte("z"), defaultExitOutputMaxBytes*4)
		got := formatExitOutput(payload, -1)
		assert.Contains(t, got, "(output:")
		assert.NotContains(t, got, "truncated")
		// Output length is unbounded; verify the full payload made it in.
		assert.Contains(t, got, string(payload[:64]))
		assert.Contains(t, got, string(payload[len(payload)-64:]))
	})

	t.Run("positive cap honors the supplied limit", func(t *testing.T) {
		// Payload of 20 bytes with a cap of 4 must truncate, reporting
		// the original 20-byte length.
		got := formatExitOutput([]byte("0123456789ABCDEFGHIJ"), 4)
		assert.Contains(t, got, `"0123"`)
		assert.Contains(t, got, "truncated from 20 bytes")
	})

	t.Run("control characters are escaped via %q", func(t *testing.T) {
		got := formatExitOutput([]byte("line1\nline2\ttab"), 0)
		// %q renders newlines and tabs as literal \n / \t escape sequences.
		assert.Contains(t, got, `\n`)
		assert.Contains(t, got, `\t`)
	})
}
