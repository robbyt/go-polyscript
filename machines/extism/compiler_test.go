package extism

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"

	extismSDK "github.com/extism/go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

// defaultCompilerOptions is a simple implementation of CompilerOptions for tests
type defaultCompilerOptions struct {
	entryPointName string
}

func (o *defaultCompilerOptions) GetEntryPointName() string {
	return o.entryPointName
}

// mockScriptReaderCloser implements io.ReadCloser for testing
type mockScriptReaderCloser struct {
	*mock.Mock
	content []byte
	offset  int
}

func newMockScriptReaderCloser(content []byte) *mockScriptReaderCloser {
	return &mockScriptReaderCloser{
		Mock:    &mock.Mock{},
		content: content,
	}
}

func (m *mockScriptReaderCloser) Read(p []byte) (n int, err error) {
	if m.offset >= len(m.content) {
		return 0, io.EOF
	}
	n = copy(p, m.content[m.offset:])
	m.offset += n
	return n, nil
}

func (m *mockScriptReaderCloser) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestCompiler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		wasmBytes  []byte
		entryPoint string
		options    *compileOptions
		validateFn func(*testing.T, *Executable)
		err        error
	}{
		{
			name:       "valid wasm binary with existing function",
			wasmBytes:  readTestWasm(t),
			entryPoint: "greet",
			validateFn: func(t *testing.T, e *Executable) {
				assert.NotNil(t, e.GetExtismByteCode())
				plugin := e.GetExtismByteCode()
				require.NotNil(t, plugin)

				// Create instance to check function existence
				instance, err := plugin.Instance(context.Background(), extismSDK.PluginInstanceConfig{})
				require.NoError(t, err)
				defer instance.Close(context.Background())

				assert.True(t, instance.FunctionExists("greet"), "Function 'greet' should exist")

				// Test function execution
				exit, output, err := instance.Call("greet", []byte(`{"input":"Test"}`))
				require.NoError(t, err)
				assert.Equal(t, uint32(0), exit)

				var result struct {
					Greeting string `json:"greeting"`
				}
				require.NoError(t, json.Unmarshal(output, &result))
				assert.Equal(t, "Hello, Test!", result.Greeting)
			},
		},
		{
			name:       "custom entry point function exists",
			wasmBytes:  readTestWasm(t),
			entryPoint: "process_complex",
			validateFn: func(t *testing.T, e *Executable) {
				assert.Equal(t, "process_complex", e.GetEntryPoint())
				plugin := e.GetExtismByteCode()
				require.NotNil(t, plugin)

				// Create instance to check function existence
				instance, err := plugin.Instance(context.Background(), extismSDK.PluginInstanceConfig{})
				require.NoError(t, err)
				defer instance.Close(context.Background())

				assert.True(t, instance.FunctionExists("process_complex"),
					"Function 'process_complex' should exist")
			},
		},
		{
			name:       "custom compilation options",
			wasmBytes:  readTestWasm(t),
			entryPoint: "greet",
			options: &compileOptions{
				EnableWASI:    true,
				RuntimeConfig: wazero.NewRuntimeConfig().WithCompilationCache(wazero.NewCompilationCache()),
			},
		},
		{
			name:      "nil content",
			wasmBytes: nil,
			err:       ErrContentNil,
		},
		{
			name:      "empty content",
			wasmBytes: []byte{},
			err:       ErrContentNil,
		},
		{
			name:      "invalid wasm binary",
			wasmBytes: []byte("not-wasm"),
			err:       ErrValidationFailed,
		},
		{
			name:       "missing function",
			wasmBytes:  readTestWasm(t),
			entryPoint: "nonexistent_function",
			err:        ErrValidationFailed,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create compiler
			compilerOptions := &defaultCompilerOptions{entryPointName: "main"}
			handler := slog.NewTextHandler(os.Stdout, nil)
			comp := NewCompiler(handler, compilerOptions)
			if tt.entryPoint != "" {
				comp.SetEntryPointName(tt.entryPoint)
			}

			// Create mock reader with content
			var reader io.ReadCloser
			if tt.wasmBytes != nil {
				reader = newMockScriptReaderCloser(tt.wasmBytes)
				reader.(*mockScriptReaderCloser).On("Close").Return(nil)
			}

			// Compile
			execContent, err := comp.Compile(reader)

			if tt.err != nil {
				require.Error(t, err)
				require.Nil(t, execContent)
				require.True(t, errors.Is(err, tt.err),
					"Expected error %v, got %v", tt.err, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, execContent)

			// Type assertion
			executable, ok := execContent.(*Executable)
			require.True(t, ok, "Expected *Executable type")

			// Validate source matches
			assert.Equal(t, tt.wasmBytes, []byte(executable.GetSource()))

			// Run custom validation if provided
			if tt.validateFn != nil {
				tt.validateFn(t, executable)
			}

			// Test Close functionality
			ctx := context.Background()
			require.NoError(t, executable.Close(ctx))

			// Verify mock expectations
			if reader != nil {
				reader.(*mockScriptReaderCloser).AssertExpectations(t)
			}
		})
	}
}
