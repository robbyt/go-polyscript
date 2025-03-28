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

	t.Run("valid wasm binary with existing function", func(t *testing.T) {
		t.Parallel()
		wasmBytes := readTestWasm(t)
		entryPoint := "greet"

		// Create compiler
		compilerOptions := &defaultCompilerOptions{entryPointName: "main"}
		handler := slog.NewTextHandler(os.Stdout, nil)
		comp := NewCompiler(handler, compilerOptions)
		comp.SetEntryPointName(entryPoint)

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

		// Create compiler
		compilerOptions := &defaultCompilerOptions{entryPointName: "main"}
		handler := slog.NewTextHandler(os.Stdout, nil)
		comp := NewCompiler(handler, compilerOptions)
		comp.SetEntryPointName(entryPoint)

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
		defer instance.Close(context.Background())

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

		// Create compiler
		compilerOptions := &defaultCompilerOptions{entryPointName: "main"}
		handler := slog.NewTextHandler(os.Stdout, nil)
		comp := NewCompiler(handler, compilerOptions)
		comp.SetEntryPointName(entryPoint)

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

		// Create compiler
		compilerOptions := &defaultCompilerOptions{entryPointName: "main"}
		handler := slog.NewTextHandler(os.Stdout, nil)
		comp := NewCompiler(handler, compilerOptions)

		// Compile with nil reader
		execContent, err := comp.Compile(nil)
		require.Error(t, err)
		require.Nil(t, execContent)
		require.True(t, errors.Is(err, ErrContentNil),
			"Expected error %v, got %v", ErrContentNil, err)
	})

	t.Run("empty content", func(t *testing.T) {
		t.Parallel()

		// Create compiler
		compilerOptions := &defaultCompilerOptions{entryPointName: "main"}
		handler := slog.NewTextHandler(os.Stdout, nil)
		comp := NewCompiler(handler, compilerOptions)

		// Create empty reader
		reader := newMockScriptReaderCloser([]byte{})
		reader.On("Close").Return(nil)

		// Compile
		execContent, err := comp.Compile(reader)
		require.Error(t, err)
		require.Nil(t, execContent)
		require.True(t, errors.Is(err, ErrContentNil),
			"Expected error %v, got %v", ErrContentNil, err)

		// Verify mock expectations
		reader.AssertExpectations(t)
	})

	t.Run("invalid wasm binary", func(t *testing.T) {
		t.Parallel()

		// Create compiler
		compilerOptions := &defaultCompilerOptions{entryPointName: "main"}
		handler := slog.NewTextHandler(os.Stdout, nil)
		comp := NewCompiler(handler, compilerOptions)

		// Create reader with invalid content
		reader := newMockScriptReaderCloser([]byte("not-wasm"))
		reader.On("Close").Return(nil)

		// Compile
		execContent, err := comp.Compile(reader)
		require.Error(t, err)
		require.Nil(t, execContent)
		require.True(t, errors.Is(err, ErrValidationFailed),
			"Expected error %v, got %v", ErrValidationFailed, err)

		// Verify mock expectations
		reader.AssertExpectations(t)
	})

	t.Run("missing function", func(t *testing.T) {
		t.Parallel()
		wasmBytes := readTestWasm(t)

		// Create compiler
		compilerOptions := &defaultCompilerOptions{entryPointName: "main"}
		handler := slog.NewTextHandler(os.Stdout, nil)
		comp := NewCompiler(handler, compilerOptions)
		comp.SetEntryPointName("nonexistent_function")

		// Create mock reader with content
		reader := newMockScriptReaderCloser(wasmBytes)
		reader.On("Close").Return(nil)

		// Compile
		execContent, err := comp.Compile(reader)
		require.Error(t, err)
		require.Nil(t, execContent)
		require.True(t, errors.Is(err, ErrValidationFailed),
			"Expected error %v, got %v", ErrValidationFailed, err)

		// Verify mock expectations
		reader.AssertExpectations(t)
	})
}
