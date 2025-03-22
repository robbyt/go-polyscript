package extism

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script"
	machineTypes "github.com/robbyt/go-polyscript/machines/types"
)

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
			t.Parallel()

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
				//nolint:staticcheck // Temporarily ignoring the "string as context key" warning until type system is fixed
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

	tests := []struct {
		name        string
		setupExe    func(t *testing.T) *script.ExecutableUnit
		wantErrType error
	}{
		{
			name: "nil bytecode",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				// Create a context provider
				ctxProvider := data.NewContextProvider(constants.EvalData)

				// Use mock content
				mockContent := &mockExecutableContent{
					machineType: machineTypes.Extism,
					source:      "invalid wasm",
					bytecode:    nil, // Nil bytecode will cause error
				}

				return &script.ExecutableUnit{
					ID:           "test-nil-bytecode",
					Content:      mockContent,
					DataProvider: ctxProvider,
				}
			},
			wantErrType: errors.New("bytecode is nil"),
		},
		{
			name: "invalid content type",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				// Create a context provider
				ctxProvider := data.NewContextProvider(constants.EvalData)

				// This is not a proper Executable
				mockContent := &mockExecutableContent{
					machineType: machineTypes.Extism,
					source:      "invalid wasm",
					bytecode:    []byte{0x00}, // Not a valid WASM module
				}

				return &script.ExecutableUnit{
					ID:           "test-invalid-content-type",
					Content:      mockContent,
					DataProvider: ctxProvider,
				}
			},
			wantErrType: errors.New("invalid executable type"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := slog.NewTextHandler(os.Stdout, nil)
			exe := tt.setupExe(t)
			evaluator := NewBytecodeEvaluator(handler, exe)

			ctx := context.Background()
			_, err := evaluator.Eval(ctx)

			if tt.wantErrType != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrType.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMarshalInputData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    map[string]any
		wantJSON bool
		wantErr  bool
	}{
		{
			name:     "empty map",
			input:    map[string]any{},
			wantJSON: false,
			wantErr:  false,
		},
		{
			name: "valid data",
			input: map[string]any{
				"str":  "value",
				"int":  42,
				"bool": true,
				"map": map[string]any{
					"nested": "data",
				},
			},
			wantJSON: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := marshalInputData(tt.input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.wantJSON {
					require.NotNil(t, result)
					require.NotEmpty(t, result)
				} else if len(tt.input) == 0 {
					require.Nil(t, result)
				}
			}
		})
	}
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

	// Create a mock executable
	exe := &script.ExecutableUnit{
		ID:           "test-basic",
		DataProvider: ctxProvider,
		Content: &mockExecutableContent{
			machineType: machineTypes.Extism,
			source:      "test wasm",
			bytecode:    []byte{0x00, 0x61, 0x73, 0x6D}, // WASM magic bytes only
		},
	}

	evaluator := NewBytecodeEvaluator(handler, exe)

	// This will fail during execution but should handle the error gracefully
	ctx := context.Background()
	evalData := map[string]any{"test": "data"}
	ctx = context.WithValue(ctx, constants.EvalData, evalData)

	_, err := evaluator.Eval(ctx)
	// We expect an error since our mock WASM isn't valid
	assert.Error(t, err)
}
