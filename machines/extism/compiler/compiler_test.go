package compiler

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

func readTestWasm(t *testing.T) []byte {
	t.Helper()
	wasmBytes, err := os.ReadFile("../testdata/examples/main.wasm")
	require.NoError(t, err)
	return wasmBytes
}

// createTestCompiler creates a compiler with the given entry point for testing
func createTestCompiler(t *testing.T, entryPoint string) *Compiler {
	t.Helper()

	comp, err := NewCompiler(
		WithEntryPoint(entryPoint),
		WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
	)
	require.NoError(t, err)
	require.NotNil(t, comp)
	return comp
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

func TestCompiler_String(t *testing.T) {
	t.Parallel()

	// Create a compiler to test the String method
	comp := createTestCompiler(t, "test_function")

	// Test String method
	result := comp.String()
	require.NotEmpty(t, result)
	require.Contains(t, result, "Compiler")
}

func TestCompiler(t *testing.T) {
	t.Parallel()

	t.Run("valid wasm binary with existing function", func(t *testing.T) {
		t.Parallel()
		wasmBytes := readTestWasm(t)
		entryPoint := "greet"

		// Create compiler using functional options
		comp := createTestCompiler(t, entryPoint)

		// Create mock reader with content
		reader := newMockScriptReaderCloser(wasmBytes)
		reader.On("Close").Return(nil)

		// Compile
		execContent, err := comp.Compile(reader)
		require.NoError(t, err)
		require.NotNil(t, execContent)

		// Type assertion
		executable, ok := execContent.(*Executable)
		require.True(t, ok, "Expected *Executable type")

		// Validate source matches
		assert.Equal(t, wasmBytes, []byte(executable.GetSource()))

		// Validate the executable
		assert.NotNil(t, executable.GetExtismByteCode())
		plugin := executable.GetExtismByteCode()
		require.NotNil(t, plugin)

		// Create instance to check function existence
		instance, err := plugin.Instance(
			context.Background(),
			extismSDK.PluginInstanceConfig{},
		)
		require.NoError(t, err)
		defer func() { require.NoError(t, instance.Close(context.Background()), "Failed to close instance") }()

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

		// Test Close functionality
		ctx := context.Background()
		require.NoError(t, executable.Close(ctx))

		// Verify mock expectations
		reader.AssertExpectations(t)
	})

	t.Run("custom entry point function exists", func(t *testing.T) {
		t.Parallel()
		wasmBytes := readTestWasm(t)
		entryPoint := "process_complex"

		// Create compiler using functional options
		comp := createTestCompiler(t, entryPoint)

		// Create mock reader with content
		reader := newMockScriptReaderCloser(wasmBytes)
		reader.On("Close").Return(nil)

		// Compile
		execContent, err := comp.Compile(reader)
		require.NoError(t, err)
		require.NotNil(t, execContent)

		// Type assertion
		executable, ok := execContent.(*Executable)
		require.True(t, ok, "Expected *Executable type")

		// Validate source matches
		assert.Equal(t, wasmBytes, []byte(executable.GetSource()))

		// Validate entry point
		assert.Equal(t, "process_complex", executable.GetEntryPoint())
		plugin := executable.GetExtismByteCode()
		require.NotNil(t, plugin)

		// Create instance to check function existence
		instance, err := plugin.Instance(
			context.Background(),
			extismSDK.PluginInstanceConfig{},
		)
		require.NoError(t, err)
		defer func() { require.NoError(t, instance.Close(context.Background()), "Failed to close instance") }()

		assert.True(t, instance.FunctionExists("process_complex"),
			"Function 'process_complex' should exist")

		// Test Close functionality
		ctx := context.Background()
		require.NoError(t, executable.Close(ctx))

		// Verify mock expectations
		reader.AssertExpectations(t)
	})

	t.Run("custom compilation options", func(t *testing.T) {
		t.Parallel()
		wasmBytes := readTestWasm(t)
		entryPoint := "greet"

		// Create compiler with custom runtime config
		comp, err := NewCompiler(
			WithEntryPoint(entryPoint),
			WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
			WithRuntimeConfig(wazero.NewRuntimeConfig()),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)

		// Create mock reader with content
		reader := newMockScriptReaderCloser(wasmBytes)
		reader.On("Close").Return(nil)

		// Compile
		execContent, err := comp.Compile(reader)
		require.NoError(t, err)
		require.NotNil(t, execContent)

		// Type assertion
		executable, ok := execContent.(*Executable)
		require.True(t, ok, "Expected *Executable type")

		// Validate source matches
		assert.Equal(t, wasmBytes, []byte(executable.GetSource()))

		// Test Close functionality
		ctx := context.Background()
		require.NoError(t, executable.Close(ctx))

		// Verify mock expectations
		reader.AssertExpectations(t)
	})

	t.Run("nil content", func(t *testing.T) {
		t.Parallel()

		comp, err := NewCompiler(
			WithEntryPoint("main"),
			WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)

		execContent, err := comp.Compile(nil)
		require.Error(t, err)
		require.Nil(t, execContent)
		require.True(t, errors.Is(err, ErrContentNil))
	})

	t.Run("empty content", func(t *testing.T) {
		t.Parallel()

		comp, err := NewCompiler(
			WithEntryPoint("main"),
			WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)

		reader := newMockScriptReaderCloser([]byte{})
		reader.On("Close").Return(nil)

		execContent, err := comp.Compile(reader)
		require.Error(t, err)
		require.Nil(t, execContent)
		require.ErrorIs(t, err, ErrContentNil)

		reader.AssertExpectations(t)
	})

	t.Run("invalid wasm binary", func(t *testing.T) {
		t.Parallel()

		comp, err := NewCompiler(
			WithEntryPoint("main"),
			WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)

		reader := newMockScriptReaderCloser([]byte("not-wasm"))
		reader.On("Close").Return(nil)

		execContent, err := comp.Compile(reader)
		require.Error(t, err)
		require.Nil(t, execContent)
		require.ErrorIs(t, err, ErrValidationFailed)

		reader.AssertExpectations(t)
	})

	t.Run("missing function", func(t *testing.T) {
		t.Parallel()
		wasmBytes := readTestWasm(t)

		comp, err := NewCompiler(
			WithEntryPoint("nonexistent_function"),
			WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)

		reader := newMockScriptReaderCloser(wasmBytes)
		reader.On("Close").Return(nil)

		execContent, err := comp.Compile(reader)
		require.Error(t, err)
		require.Nil(t, execContent)
		require.ErrorIs(t, err, ErrValidationFailed)

		reader.AssertExpectations(t)
	})
}
