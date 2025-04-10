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

func TestLoadInputData(t *testing.T) {
	t.Parallel()

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)

			// Create a context provider
			ctxProvider := data.NewContextProvider(constants.EvalData)

			// Create a dummy executableUnit
			dummyExe := &script.ExecutableUnit{
				DataProvider: ctxProvider,
			}

			evaluator := NewBytecodeEvaluator(handler, dummyExe)
			ctx := context.Background()

			if tt.ctxData != nil {
				// Temporarily ignoring the "string as context key" warning until type system is fixed
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
}

func TestBytecodeEvaluatorInvalidInputs(t *testing.T) {
	t.Parallel()

	// Common test setup helper
	setupTest := func(content *mockExecutableContent) (slog.Handler, *script.ExecutableUnit) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		ctxProvider := data.NewContextProvider(constants.EvalData)

		exe := &script.ExecutableUnit{
			ID:           "test-case",
			Content:      content,
			DataProvider: ctxProvider,
		}

		return handler, exe
	}

	// Test case: nil bytecode
	t.Run("nil bytecode", func(t *testing.T) {
		mockContent := &mockExecutableContent{
			machineType: machineTypes.Extism,
			source:      "invalid wasm",
			bytecode:    nil, // Nil bytecode will cause error
		}

		handler, exe := setupTest(mockContent)
		evaluator := NewBytecodeEvaluator(handler, exe)

		ctx := context.Background()
		_, err := evaluator.Eval(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "bytecode is nil")
	})

	// Test case: invalid content type
	t.Run("invalid content type", func(t *testing.T) {
		mockContent := &mockExecutableContent{
			machineType: machineTypes.Extism,
			source:      "invalid wasm",
			bytecode:    []byte{0x00}, // Not a valid WASM module
		}

		handler, exe := setupTest(mockContent)
		evaluator := NewBytecodeEvaluator(handler, exe)

		ctx := context.Background()
		_, err := evaluator.Eval(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid executable type")
	})
}

func TestNilHandlerFallback(t *testing.T) {
	// Test that the evaluator handles nil handlers by creating a default

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
	evaluator := NewBytecodeEvaluator(nil, exe)

	// Shouldn't panic
	require.NotNil(t, evaluator)
	require.NotNil(t, evaluator.logger)
	require.NotNil(t, evaluator.logHandler)
}

func TestEvaluatorString(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	evaluator := NewBytecodeEvaluator(handler, nil)

	// Test the string representation
	strRep := evaluator.String()
	require.Equal(t, "extism.BytecodeEvaluator", strRep)
}

func TestEvalWithNilExecutableUnit(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	evaluator := NewBytecodeEvaluator(handler, nil)

	// Attempt to evaluate with nil executable unit
	ctx := context.Background()
	_, err := evaluator.Eval(ctx)

	// Should get an error
	require.Error(t, err)
	require.Contains(t, err.Error(), "executable unit is nil")
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

// TestBasicExecution is a simplified test that mocks the execution
func TestBasicExecution(t *testing.T) {
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
		exitCode: 1, // Will cause an error
		output:   []byte(`{"error":"something went wrong"}`),
	}
	mockPlugin.On("Instance", mock.Anything, mock.Anything).Return(mockInstance, nil)
	mockPlugin.On("Close", mock.Anything).Return(nil)

	// Create a real compiler.Executable with our mock plugin
	content := createMockExecutable(mockPlugin, "main")

	// Create a mock executable
	exe := &script.ExecutableUnit{
		ID:           "test-basic",
		DataProvider: ctxProvider,
		Content:      content,
	}

	evaluator := NewBytecodeEvaluator(handler, exe)

	// This will fail during execution but should handle the error gracefully
	ctx := context.Background()
	evalData := map[string]any{"test": "data"}
	ctx = context.WithValue(ctx, constants.EvalData, evalData)

	_, err := evaluator.Eval(ctx)
	// We expect an error since our mock returns an error
	assert.Error(t, err)
}

func TestPrepareContext(t *testing.T) {
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			exe := tt.setupExe(t)
			evaluator := NewBytecodeEvaluator(handler, exe)

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

func TestExecHelper(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func() (*mockPluginInstance, context.Context, context.CancelFunc)
		entryPoint  string
		input       []byte
		wantErr     bool
		errContains string
	}{
		{
			name: "successful execution with json output",
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
			name: "successful execution with string output",
			setup: func() (*mockPluginInstance, context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				return &mockPluginInstance{
					exitCode: 0,
					output:   []byte(`plain text output`),
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockInstance, ctx, cancel := tt.setup()
			defer cancel()

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			result, execTime, err := execHelper(ctx, logger, mockInstance, tt.entryPoint, tt.input)

			// Verify the mock was called
			assert.True(t, mockInstance.wasCalled, "Expected the mock instance to be called")

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
}

/*
func TestEvalWithCancelledContext(t *testing.T) {
	// Load the test WASM file
	wasmContent, err := os.ReadFile(testWasmPath)
	require.NoError(t, err, "Failed to read WASM test file")

	// Create a temporary directory using the testing package
	tmpDir := t.TempDir()

	// Write the test WASM bytes to a file
	wasmFile := filepath.Join(tmpDir, "test.wasm")
	err = os.WriteFile(wasmFile, wasmContent, 0o644)
	require.NoError(t, err, "Failed to write test WASM file")

	// Create a mock compiled plugin
	ctx := context.Background()
	compileOpts := compile.WithDefaultCompileSettings()
	compiledPlugin, err := compile.CompileBytes(ctx, wasmContent, compileOpts)
	require.NoError(t, err, "Failed to compile plugin")

	// Create our executable
	exec := compile.NewExecutable(wasmContent, compiledPlugin, "greet")

	// Create a context provider
	ctxProvider := data.NewContextProvider(constants.EvalData)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the executable unit
	execUnit := &script.ExecutableUnit{
		ID:           "test-cancellation",
		DataProvider: ctxProvider,
		Content:      exec,
	}

	// Create handler and evaluator
	handler := slog.NewTextHandler(os.Stdout, nil)
	evaluator := NewBytecodeEvaluator(handler, execUnit)

	// Set up context data
	evalData := map[string]any{"name": "TestUser"}
	ctx = context.WithValue(ctx, constants.EvalData, evalData)

	// Cancel the context before evaluation
	cancel()

	// Try to evaluate with the cancelled context
	_, err = evaluator.Eval(ctx)

	// Should get an error (either cancellation or plugin error)
	require.Error(t, err)
}
*/

/*
// TestStaticAndDynamicDataCombination tests how static data and dynamic data are combined
// with the CompositeProvider
func TestStaticAndDynamicDataCombination(t *testing.T) {
	t.Skip("Need to confirm behavior of the input_data in ctx")
	// Load the test WASM file
	wasmContent, err := os.ReadFile(testWasmPath)
	require.NoError(t, err, "Failed to read WASM test file")

	// Create a mock compiled plugin
	ctx := context.Background()
	compileOpts := compile.WithDefaultCompileSettings()
	compiledPlugin, err := compile.CompileBytes(ctx, wasmContent, compileOpts)
	require.NoError(t, err, "Failed to compile plugin")

	// Create our executable
	exec := compiler.NewExecutable(wasmContent, compiledPlugin, "greet")

	// Create a context provider for runtime data
	ctxProvider := data.NewContextProvider(constants.EvalData)

	// Create static data for compile-time configuration
	staticData := map[string]any{"initial": "value"}

	// Create a static provider
	staticProvider := data.NewStaticProvider(staticData)

	// Create a composite provider that combines static and context data
	compositeProvider := data.NewCompositeProvider(staticProvider, ctxProvider)

	// Create the executable unit with the composite provider
	execUnit := &script.ExecutableUnit{
		ID:           "test-data-provider",
		DataProvider: compositeProvider,
		Content:      exec,
	}

	// Create handler and evaluator
	handler := slog.NewTextHandler(os.Stdout, nil)
	evaluator := NewBytecodeEvaluator(handler, execUnit)

	// Create a context
	ctx = context.Background()

	// First test: load data with empty context
	result1, err := evaluator.loadInputData(ctx)
	require.NoError(t, err)
	assert.Contains(t, result1, "initial")
	assert.Equal(t, "value", result1["initial"])

	// Second test: add data to context and verify it's merged with static data
	inputData := map[string]any{"input": "test input"}
	enrichedCtx, err := evaluator.PrepareContext(ctx, inputData)
	require.NoError(t, err)

	result2, err := evaluator.loadInputData(enrichedCtx)
	require.NoError(t, err)

	// Static data should still be there at top level
	assert.Contains(t, result2, "initial")
	assert.Equal(t, "value", result2["initial"])

	// Runtime data from the ContextProvider is stored under the 'input_data' key
	assert.Contains(t, result2, constants.InputData)

	// Extract the input_data map and verify it's the correct type
	dynamicData, ok := result2[constants.InputData].(map[string]any)
	require.True(t, ok, "input_data should be a map")

	// Verify our input data was correctly stored in the input_data map
	assert.Contains(t, dynamicData, "input")
	assert.Equal(t, "test input", dynamicData["input"])
}
*/

// TestBytecodeEvaluator_Cancel tests the behavior when a context is cancelled
func TestBytecodeEvaluator_Cancel(t *testing.T) {
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

	evaluator := NewBytecodeEvaluator(handler, execUnit)

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
}

// TestBytecodeEvaluator_Exec tests the exec method directly
func TestBytecodeEvaluator_Exec(t *testing.T) {
	// Define test cases
	tests := []struct {
		name          string
		setupMocks    func() (*MockCompiledPlugin, *mockPluginInstance)
		entryPoint    string
		inputData     map[string]any
		expectedValue any
		expectError   bool
		errorContains string
	}{
		{
			name: "successful execution",
			setupMocks: func() (*MockCompiledPlugin, *mockPluginInstance) {
				mockPlugin := new(MockCompiledPlugin)
				mockInstance := &mockPluginInstance{
					exitCode: 0,
					output:   []byte(`{"result":"success"}`),
				}
				mockPlugin.On("Instance", mock.Anything, mock.Anything).Return(mockInstance, nil)
				mockPlugin.On("Close", mock.Anything).Return(nil)
				return mockPlugin, mockInstance
			},
			entryPoint:    "main",
			inputData:     map[string]any{"test": "data"},
			expectedValue: map[string]any{"result": "success"},
			expectError:   false,
		},
		{
			name: "error creating instance",
			setupMocks: func() (*MockCompiledPlugin, *mockPluginInstance) {
				mockPlugin := new(MockCompiledPlugin)
				mockInstance := &mockPluginInstance{
					wasClosed: true, // Mock as if it was closed since the exec method closes the instance on error
				}
				mockPlugin.On("Instance", mock.Anything, mock.Anything).
					Return(mockInstance, errors.New("instance creation error"))
				mockPlugin.On("Close", mock.Anything).Return(nil)
				return mockPlugin, mockInstance
			},
			entryPoint:    "main",
			inputData:     map[string]any{"test": "data"},
			expectError:   true,
			errorContains: "failed to create plugin instance",
		},
		{
			name: "execution error",
			setupMocks: func() (*MockCompiledPlugin, *mockPluginInstance) {
				mockPlugin := new(MockCompiledPlugin)
				mockInstance := &mockPluginInstance{
					callErr: errors.New("execution error"),
				}
				mockPlugin.On("Instance", mock.Anything, mock.Anything).Return(mockInstance, nil)
				mockPlugin.On("Close", mock.Anything).Return(nil)
				return mockPlugin, mockInstance
			},
			entryPoint:    "main",
			inputData:     map[string]any{"test": "data"},
			expectError:   true,
			errorContains: "execution error",
		},
		{
			name: "non-zero exit code",
			setupMocks: func() (*MockCompiledPlugin, *mockPluginInstance) {
				mockPlugin := new(MockCompiledPlugin)
				mockInstance := &mockPluginInstance{
					exitCode: 1,
					output:   []byte(`{"error":"something went wrong"}`),
				}
				mockPlugin.On("Instance", mock.Anything, mock.Anything).Return(mockInstance, nil)
				mockPlugin.On("Close", mock.Anything).Return(nil)
				return mockPlugin, mockInstance
			},
			entryPoint:    "main",
			inputData:     map[string]any{"test": "data"},
			expectError:   true,
			errorContains: "non-zero exit code",
		},
		{
			name: "context cancellation",
			setupMocks: func() (*MockCompiledPlugin, *mockPluginInstance) {
				mockPlugin := new(MockCompiledPlugin)
				mockInstance := &mockPluginInstance{
					callErr: context.Canceled,
				}
				mockPlugin.On("Instance", mock.Anything, mock.Anything).Return(mockInstance, nil)
				mockPlugin.On("Close", mock.Anything).Return(nil)
				return mockPlugin, mockInstance
			},
			entryPoint:    "main",
			inputData:     map[string]any{"test": "data"},
			expectError:   true,
			errorContains: "execution",
		},
		{
			name: "close error (should still succeed)",
			setupMocks: func() (*MockCompiledPlugin, *mockPluginInstance) {
				mockPlugin := new(MockCompiledPlugin)
				mockInstance := &mockPluginInstance{
					exitCode: 0,
					output:   []byte(`{"result":"success"}`),
					closeErr: errors.New("close error"),
				}
				mockPlugin.On("Instance", mock.Anything, mock.Anything).Return(mockInstance, nil)
				mockPlugin.On("Close", mock.Anything).Return(nil)
				return mockPlugin, mockInstance
			},
			entryPoint:    "main",
			inputData:     map[string]any{"test": "data"},
			expectedValue: map[string]any{"result": "success"},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockPlugin, mockInstance := tt.setupMocks()

			// Create handler and evaluator
			handler := slog.NewTextHandler(os.Stdout, nil)

			// Create the unit under test
			evaluator := NewBytecodeEvaluator(handler, nil)

			// Convert input data to JSON
			inputJSON, err := mockLoadInputData(tt.inputData)
			require.NoError(t, err)

			// Call the exec method directly
			result, err := evaluator.exec(
				context.Background(),
				mockPlugin,
				tt.entryPoint,
				adapters.NewPluginInstanceConfig(),
				inputJSON,
			)

			// Verify instance was called
			mockPlugin.AssertCalled(t, "Instance", mock.Anything, mock.Anything)

			// Check for expected errors
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Check result
				resultValue := result.Interface()
				assert.Equal(t, tt.expectedValue, resultValue)
			}

			// Close should always be called
			assert.True(t, mockInstance.wasClosed, "Instance should be closed")
		})
	}
}

// TestExtismDirectInputFormat tests how input data is formatted for Extism
func TestExtismDirectInputFormat(t *testing.T) {
	// Create a test map that simulates data from our providers
	inputData := map[string]any{
		"initial": "top-level-value", // Static data at top level
		"input_data": map[string]any{ // Dynamic data nested under input_data
			"input":   "API User",
			"request": map[string]any{}, // HTTP request data nested under input_data
		},
	}

	// First, log the structure to understand what we're dealing with
	t.Logf("Input data structure: %#v", inputData)

	// Convert the input data for Extism
	jsonBytes, err := internal.ConvertToExtismFormat(inputData)
	require.NoError(t, err)
	require.NotNil(t, jsonBytes)

	// Log the JSON output
	t.Logf("JSON for Extism: %s", string(jsonBytes))

	// Verify current behavior
	expected := `{"initial":"top-level-value","input_data":{"input":"API User","request":{}}}`
	assert.JSONEq(t, expected, string(jsonBytes))
}

// Helper function to create mock input data JSON
func mockLoadInputData(data map[string]any) ([]byte, error) {
	if data == nil {
		return []byte("{}"), nil
	}
	return internal.ConvertToExtismFormat(data)
}
