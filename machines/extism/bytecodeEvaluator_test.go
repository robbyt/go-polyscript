package extism

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script"
	machineTypes "github.com/robbyt/go-polyscript/machines/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var emptyScriptData = make(map[string]any)

// mockExecutableContent is a mock implementation of script.ExecutableContent
type mockExecutableContent struct {
	mock.Mock
}

func (m *mockExecutableContent) GetSource() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockExecutableContent) GetByteCode() any {
	args := m.Called()
	return args.Get(0)
}

func (m *mockExecutableContent) GetMachineType() machineTypes.Type {
	args := m.Called()
	return args.Get(0).(machineTypes.Type)
}

func (m *mockExecutableContent) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestMarshalInputData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		inputData map[string]any
		wantErr   bool
		wantNil   bool
	}{
		{
			name:      "empty map",
			inputData: map[string]any{},
			wantNil:   true,
		},
		{
			name: "simple map",
			inputData: map[string]any{
				"key": "value",
			},
			wantNil: false,
		},
		{
			name: "complex map",
			inputData: map[string]any{
				"string": "value",
				"int":    42,
				"bool":   true,
				"nested": map[string]any{
					"key": "nested value",
				},
				"array": []string{"one", "two"},
			},
			wantNil: false,
		},
		{
			name: "map with channel (should error)",
			inputData: map[string]any{
				"channel": make(chan int),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := marshalInputData(tt.inputData)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				// Verify it's valid JSON
				var decoded map[string]any
				err = json.Unmarshal(result, &decoded)
				require.NoError(t, err)
				assert.Equal(t, len(tt.inputData), len(decoded))
			}
		})
	}
}

func TestLoadInputDataFromCtx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		ctxData       any
		expectedEmpty bool
	}{
		{
			name:          "nil context data",
			ctxData:       nil,
			expectedEmpty: true,
		},
		{
			name: "valid map data",
			ctxData: map[string]any{
				"key": "value",
			},
			expectedEmpty: false,
		},
		{
			name:          "wrong type (string)",
			ctxData:       "not a map",
			expectedEmpty: true,
		},
		{
			name:          "wrong type (int)",
			ctxData:       42,
			expectedEmpty: true,
		},
		{
			name:          "wrong map type",
			ctxData:       map[int]string{1: "value"},
			expectedEmpty: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := slog.NewTextHandler(os.Stdout, nil)
			evaluator := NewBytecodeEvaluator(handler, nil)
			ctx := context.Background()

			if tt.ctxData != nil {
				//nolint:staticcheck // Temporarily ignoring the "string as context key" warning until type system is fixed
				ctx = context.WithValue(ctx, constants.EvalData, tt.ctxData)
			}

			// Note: We're still testing the old method during the transition,
			// but in production we'll use the new dataProvider.GetInputData()
			result := evaluator.loadInputDataFromCtx(ctx)

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

	tests := []struct {
		name        string
		setupExe    func(t *testing.T) *script.ExecutableUnit
		wantErrType error
	}{
		{
			name: "nil executable",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				return nil
			},
			wantErrType: errors.New("executable unit is nil"),
		},
		{
			name: "nil content",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				return &script.ExecutableUnit{
					ID:         "test",
					Content:    nil,
					ScriptData: emptyScriptData,
				}
			},
			wantErrType: errors.New("content is nil"),
		},
		{
			name: "wrong content type",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				mockContent := &mockExecutableContent{}
				mockContent.On("GetByteCode").Return([]byte("mockbytecode"))
				mockContent.On("GetMachineType").Return(machineTypes.Type("mock"))
				return &script.ExecutableUnit{
					ID:         "test",
					Content:    mockContent,
					ScriptData: emptyScriptData,
				}
			},
			wantErrType: errors.New("invalid executable type"),
		},
		{
			name: "empty execution ID",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				wasmBytes := readTestWasm(t)
				plugin, err := CompileBytes(context.Background(), wasmBytes, nil)
				require.NoError(t, err)

				executable := NewExecutable(wasmBytes, plugin, "run")

				return &script.ExecutableUnit{
					ID:         "", // Empty ID
					Content:    executable,
					ScriptData: emptyScriptData,
				}
			},
			wantErrType: errors.New("execution ID is empty"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := slog.NewTextHandler(os.Stdout, nil)
			evaluator := NewBytecodeEvaluator(handler, nil)
			ctx := context.Background()

			exe := tt.setupExe(t)
			_, err := evaluator.Eval(ctx, exe)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrType.Error())
		})
	}
}

// This function would test additional error handling in the exec function, but
// because the Extism SDK uses concrete types rather than interfaces, mocking is difficult.
// TODO: Consider refactoring the code to use interfaces for easier testing, or
// set up an integration test that can deliberately cause errors.
func TestValidScript(t *testing.T) {
	t.Parallel()
	// Setup logging
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(handler))

	// Read test WASM binary
	wasmBytes := readTestWasm(t)
	require.NotEmpty(t, wasmBytes)

	// Create helper to build test executables
	evalBuilder := func(t *testing.T, funcName string) (*script.ExecutableUnit, *BytecodeEvaluator) {
		t.Helper()

		// Create compiler with new reader each time
		// Create compiler options
		compilerOptions := &defaultCompilerOptions{entryPointName: funcName}
		handler := slog.NewTextHandler(os.Stdout, nil)
		compiler := NewCompiler(handler, compilerOptions)
		compiler.SetEntryPointName(funcName)

		// Convert bytes to reader
		reader := newMockScriptReaderCloser(wasmBytes)
		reader.On("Close").Return(nil)

		content, err := compiler.Compile(reader)
		require.NoError(t, err)
		require.NotNil(t, content)

		exe := &script.ExecutableUnit{
			ID:         t.Name(),
			Content:    content,
			ScriptData: emptyScriptData,
		}
		require.NotNil(t, exe)

		handlerEval := slog.NewTextHandler(os.Stdout, nil)
		evaluator := NewBytecodeEvaluator(handlerEval, nil)
		require.NotNil(t, evaluator)

		return exe, evaluator
	}

	t.Run("basic functions", func(t *testing.T) {
		tests := []struct {
			name          string
			function      string
			input         []byte
			expected      string
			expectedType  data.Types
			expectedValue any
		}{
			{
				name:          "greet function",
				function:      "greet",
				input:         []byte("World"),
				expected:      "Hello, World!",
				expectedType:  data.MAP,
				expectedValue: map[string]any{"greeting": "Hello, World!"},
			},
			{
				name:          "reverse string",
				function:      "reverse_string",
				input:         []byte("Hello"),
				expected:      "olleH",
				expectedType:  data.MAP,
				expectedValue: map[string]any{"reversed": "olleH"},
			},
			{
				name:         "count vowels",
				function:     "count_vowels",
				input:        []byte("Hello World"),
				expectedType: data.MAP,
				// Result will be JSON object
				expected: `{"count":3,"vowels":"aeiouAEIOU","input":"Hello World"}`,
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				exe, evaluator := evalBuilder(t, tt.function)

				// Create context with input data
				inputData := map[string]any{
					"input": string(tt.input),
				}
				//nolint:staticcheck // Temporarily ignoring the "string as context key" warning until type system is fixed
				ctx := context.WithValue(context.Background(), constants.EvalData, inputData)

				response, err := evaluator.Eval(ctx, exe)
				require.NoError(t, err)
				require.NotNil(t, response)

				assert.Equal(t, tt.expectedType, response.Type())
				if tt.expectedValue != nil {
					// Direct comparison - response.Interface() already returns any
					assert.Equal(t, tt.expectedValue, response.Interface())
				}
			})
		}
	})

	t.Run("complex processing", func(t *testing.T) {
		exe, evaluator := evalBuilder(t, "process_complex")

		request := map[string]any{
			"id":        "test-123",
			"timestamp": time.Now().Unix(),
			"data": map[string]any{
				"key1": "value1",
				"key2": 42,
			},
			"tags": []string{"test", "example"},
			"metadata": map[string]string{
				"source":  "unit-test",
				"version": "1.0",
			},
			"count":  42,
			"active": true,
		}

		//nolint:staticcheck // Temporarily ignoring the "string as context key" warning until type system is fixed
		ctx := context.WithValue(context.Background(), constants.EvalData, request)

		response, err := evaluator.Eval(ctx, exe)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.Equal(t, data.MAP, response.Type())

		result := response.Interface().(map[string]any)
		assert.Equal(t, "test-123", result["request_id"])
		assert.Equal(t, 2, result["tag_count"])
		assert.Equal(t, 2, result["meta_count"])
		assert.Equal(t, true, result["is_active"])
	})
}
