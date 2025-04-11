package evaluator

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/machines/extism/adapters"
	"github.com/robbyt/go-polyscript/machines/extism/compiler"
	"github.com/robbyt/go-polyscript/machines/extism/internal"
	machineTypes "github.com/robbyt/go-polyscript/machines/types"
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
	data ...any,
) (context.Context, error) {
	return ctx, m.err
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
	machineType machineTypes.Type
	source      string
	bytecode    any
}

func (m *mockExecutableContent) GetMachineType() machineTypes.Type {
	return m.machineType
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

			ctx := context.Background()
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
			require.Equal(t, float64(42), resultMap["value"])
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
			ctx := context.Background()
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
					ctx := context.Background()

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

			ctx := context.Background()
			_, err := evaluator.Eval(ctx)

			require.Error(t, err)
			require.Contains(t, err.Error(), "executable unit is nil")
		})

		// Test nil bytecode
		t.Run("nil bytecode", func(t *testing.T) {
			mockContent := &mockExecutableContent{
				machineType: machineTypes.Extism,
				source:      "invalid wasm",
				bytecode:    nil, // Nil bytecode will cause error
			}

			handler := slog.NewTextHandler(os.Stdout, nil)
			ctxProvider := data.NewContextProvider(constants.EvalData)

			exe := &script.ExecutableUnit{
				ID:           "test-case",
				Content:      mockContent,
				DataProvider: ctxProvider,
			}

			evaluator := New(handler, exe)

			ctx := context.Background()
			_, err := evaluator.Eval(ctx)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "bytecode is nil")
		})

		// Test invalid content type
		t.Run("invalid content type", func(t *testing.T) {
			mockContent := &mockExecutableContent{
				machineType: machineTypes.Extism,
				source:      "invalid wasm",
				bytecode:    []byte{0x00}, // Not a valid WASM plugin
			}

			handler := slog.NewTextHandler(os.Stdout, nil)
			ctxProvider := data.NewContextProvider(constants.EvalData)

			exe := &script.ExecutableUnit{
				ID:           "test-case",
				Content:      mockContent,
				DataProvider: ctxProvider,
			}

			evaluator := New(handler, exe)

			ctx := context.Background()
			_, err := evaluator.Eval(ctx)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid executable type")
		})

		// Test context cancellation
		t.Run("context cancellation", func(t *testing.T) {
			// Create a cancel context
			ctx, cancel := context.WithCancel(context.Background())

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

			// Should get a cancellation error
			assert.Error(t, err)
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
			ctx := context.Background()
			evalData := map[string]any{"test": "data"}
			ctx = context.WithValue(ctx, constants.EvalData, evalData)

			_, err := evaluator.Eval(ctx)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "non-zero exit code")
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
			ctx := context.Background()

			_, err := evaluator.Eval(ctx)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create plugin instance")
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
				name        string
				setup       func() (*mockPluginInstance, context.Context, context.CancelFunc)
				entryPoint  string
				input       []byte
				wantErr     bool
				errContains string
			}{
				{
					name: "successful execution",
					setup: func() (*mockPluginInstance, context.Context, context.CancelFunc) {
						ctx, cancel := context.WithCancel(context.Background())
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
						ctx, cancel := context.WithCancel(context.Background())
						return &mockPluginInstance{
							exitCode: 1,
							output:   []byte(`{"error": "something went wrong"}`),
						}, ctx, cancel
					},
					entryPoint:  "main",
					input:       []byte(`{"key":"value"}`),
					wantErr:     true,
					errContains: "non-zero exit code",
				},
				{
					name: "execution error",
					setup: func() (*mockPluginInstance, context.Context, context.CancelFunc) {
						ctx, cancel := context.WithCancel(context.Background())
						return &mockPluginInstance{
							callErr: errors.New("execution failed"),
						}, ctx, cancel
					},
					entryPoint:  "main",
					input:       []byte(`{"key":"value"}`),
					wantErr:     true,
					errContains: "execution failed",
				},
				{
					name: "context cancellation",
					setup: func() (*mockPluginInstance, context.Context, context.CancelFunc) {
						ctx, cancel := context.WithCancel(context.Background())
						mock := &mockPluginInstance{
							cancelFunc: cancel, // This will cancel the context during execution
							callErr:    context.Canceled,
						}
						return mock, ctx, cancel
					},
					entryPoint:  "main",
					input:       []byte(`{"key":"value"}`),
					wantErr:     true,
					errContains: "cancelled",
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
					)

					// Verify the mock was called
					assert.True(
						t,
						mockInstance.wasCalled,
						"Expected the mock instance to be called",
					)

					// Check for expected errors
					if tt.wantErr {
						assert.Error(t, err)
						if tt.errContains != "" {
							assert.Contains(t, err.Error(), tt.errContains)
						}
					} else {
						assert.NoError(t, err)
						assert.NotNil(t, result)
					}

					// Execution time should always be measured
					assert.Greater(t, execTime.Nanoseconds(), int64(0))
				})
			}
		})
	})
}

// TestEvaluator_PrepareContext tests the PrepareContext method with various scenarios
func TestEvaluator_PrepareContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupExe    func(t *testing.T) *script.ExecutableUnit
		inputs      []any
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
			inputs:      []any{map[string]any{"test": "data"}},
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
			inputs:    []any{map[string]any{"test": "data"}},
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
			inputs:    []any{},
			wantError: false,
		},
		{
			name: "nil executable unit",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()
				return nil
			},
			inputs:      []any{map[string]any{"test": "data"}},
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
			inputs:      []any{map[string]any{"test": "data"}},
			wantError:   true,
			expectedErr: "provider error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			exe := tt.setupExe(t)
			evaluator := New(handler, exe)

			ctx := context.Background()
			enrichedCtx, err := evaluator.PrepareContext(ctx, tt.inputs...)

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
