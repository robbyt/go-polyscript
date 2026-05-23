package compiler

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/engines/extism/adapters"
	"github.com/robbyt/go-polyscript/engines/extism/compiler/internal/compile"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

// createTestCompiler creates a compiler with the given entry point for testing
func createTestCompiler(t *testing.T, entryPoint string) *Compiler {
	t.Helper()

	comp, err := New(
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

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("basic creation", func(t *testing.T) {
		comp, err := New(
			WithEntryPoint("main"),
			WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)

		// Test String method
		result := comp.String()
		require.NotEmpty(t, result)
		require.Contains(t, result, "Compiler")
	})

	t.Run("with entry point", func(t *testing.T) {
		comp, err := New(
			WithEntryPoint("custom_function"),
			WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)
	})

	t.Run("with custom runtime config", func(t *testing.T) {
		comp, err := New(
			WithEntryPoint("main"),
			WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
			WithRuntimeConfig(wazero.NewRuntimeConfig()),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)
	})

	t.Run("with custom logger", func(t *testing.T) {
		handler := slog.NewTextHandler(io.Discard, nil)
		logger := slog.New(handler)
		comp, err := New(
			WithEntryPoint("main"),
			WithLogger(logger),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)
	})
}

func TestCompiler_Compile(t *testing.T) {
	t.Parallel()

	t.Run("success cases", func(t *testing.T) {
		t.Run("valid wasm binary with existing function", func(t *testing.T) {
			wasmBytes := wasmdata.TestModule
			entryPoint := "greet"
			comp := createTestCompiler(t, entryPoint)
			reader := newMockScriptReaderCloser(wasmBytes)
			reader.On("Close").Return(nil)

			execContent, err := comp.Compile(t.Context(), reader)
			require.NoError(t, err)
			require.NotNil(t, execContent)

			executable, ok := execContent.(*Executable)
			require.True(t, ok, "Expected *Executable type")
			assert.Equal(t, wasmBytes, []byte(executable.GetSource()))

			plugin := executable.GetExtismByteCode()
			require.NotNil(t, plugin)

			instance, err := plugin.Instance(
				t.Context(),
				extismSDK.PluginInstanceConfig{},
			)
			require.NoError(t, err)
			defer func() {
				require.NoError(t, instance.Close(t.Context()), "Failed to close instance")
			}()

			assert.True(t, instance.FunctionExists("greet"), "Function 'greet' should exist")

			exit, output, err := instance.Call("greet", []byte(`{"input":"Test"}`))
			require.NoError(t, err)
			assert.Equal(t, uint32(0), exit)

			var result struct {
				Greeting string `json:"greeting"`
			}
			require.NoError(t, json.Unmarshal(output, &result))
			assert.Equal(t, "Hello, Test!", result.Greeting)

			ctx := t.Context()
			require.NoError(t, executable.Close(ctx))
			reader.AssertExpectations(t)
		})

		t.Run("custom entry point function exists", func(t *testing.T) {
			wasmBytes := wasmdata.TestModule
			entryPoint := "process_complex"
			comp := createTestCompiler(t, entryPoint)
			reader := newMockScriptReaderCloser(wasmBytes)
			reader.On("Close").Return(nil)

			execContent, err := comp.Compile(t.Context(), reader)
			require.NoError(t, err)
			require.NotNil(t, execContent)

			executable, ok := execContent.(*Executable)
			require.True(t, ok, "Expected *Executable type")
			assert.Equal(t, wasmBytes, []byte(executable.GetSource()))
			assert.Equal(t, "process_complex", executable.GetEntryPoint())

			plugin := executable.GetExtismByteCode()
			require.NotNil(t, plugin)

			instance, err := plugin.Instance(
				t.Context(),
				extismSDK.PluginInstanceConfig{},
			)
			require.NoError(t, err)
			defer func() {
				require.NoError(t, instance.Close(t.Context()), "Failed to close instance")
			}()

			assert.True(t, instance.FunctionExists("process_complex"),
				"Function 'process_complex' should exist")

			ctx := t.Context()
			require.NoError(t, executable.Close(ctx))
			reader.AssertExpectations(t)
		})

		t.Run("custom compilation options", func(t *testing.T) {
			wasmBytes := wasmdata.TestModule
			entryPoint := "greet"

			comp, err := New(
				WithEntryPoint(entryPoint),
				WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
				WithRuntimeConfig(wazero.NewRuntimeConfig()),
			)
			require.NoError(t, err)
			require.NotNil(t, comp)

			reader := newMockScriptReaderCloser(wasmBytes)
			reader.On("Close").Return(nil)

			execContent, err := comp.Compile(t.Context(), reader)
			require.NoError(t, err)
			require.NotNil(t, execContent)

			executable, ok := execContent.(*Executable)
			require.True(t, ok, "Expected *Executable type")
			assert.Equal(t, wasmBytes, []byte(executable.GetSource()))

			ctx := t.Context()
			require.NoError(t, executable.Close(ctx))
			reader.AssertExpectations(t)
		})
	})

	t.Run("error cases", func(t *testing.T) {
		t.Run("nil content", func(t *testing.T) {
			comp, err := New(
				WithEntryPoint("main"),
				WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
			)
			require.NoError(t, err)
			require.NotNil(t, comp)

			execContent, err := comp.Compile(t.Context(), nil)
			require.Error(t, err)
			require.Nil(t, execContent)
			require.ErrorIs(t, err, ErrContentNil)
		})

		t.Run("empty content", func(t *testing.T) {
			comp, err := New(
				WithEntryPoint("main"),
				WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
			)
			require.NoError(t, err)
			require.NotNil(t, comp)

			reader := newMockScriptReaderCloser([]byte{})
			reader.On("Close").Return(nil)

			execContent, err := comp.Compile(t.Context(), reader)
			require.Error(t, err)
			require.Nil(t, execContent)
			require.ErrorIs(t, err, ErrContentNil)

			reader.AssertExpectations(t)
		})

		t.Run("invalid wasm binary", func(t *testing.T) {
			comp, err := New(
				WithEntryPoint("main"),
				WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
			)
			require.NoError(t, err)
			require.NotNil(t, comp)

			reader := newMockScriptReaderCloser([]byte("not-wasm"))
			reader.On("Close").Return(nil)

			execContent, err := comp.Compile(t.Context(), reader)
			require.Error(t, err)
			require.Nil(t, execContent)
			require.ErrorIs(t, err, ErrValidationFailed)

			reader.AssertExpectations(t)
		})

		t.Run("missing function", func(t *testing.T) {
			wasmBytes := wasmdata.TestModule
			comp, err := New(
				WithEntryPoint("nonexistent_function"),
				WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
			)
			require.NoError(t, err)
			require.NotNil(t, comp)

			reader := newMockScriptReaderCloser(wasmBytes)
			reader.On("Close").Return(nil)

			execContent, err := comp.Compile(t.Context(), reader)
			require.Error(t, err)
			require.Nil(t, execContent)
			require.ErrorIs(t, err, ErrValidationFailed)

			reader.AssertExpectations(t)
		})
	})
}

// TestCompiler_Compile_Branches exercises the Compile() error paths that are not
// reachable through the real Extism SDK with a valid module: io.ReadAll failure,
// reader Close failure, compile-yields-nil-plugin, plugin.Instance failure, and
// instance.Close-on-defer failure. Uses the unexported compileFn seam.
func TestCompiler_Compile_Branches(t *testing.T) {
	t.Parallel()

	t.Run("io.ReadAll fails", func(t *testing.T) {
		t.Parallel()
		comp := createTestCompiler(t, "main")
		reader := &errReader{readErr: errors.New("read kaboom")}

		execContent, err := comp.Compile(t.Context(), reader)
		require.Error(t, err)
		require.Nil(t, execContent)
		require.ErrorContains(t, err, "failed to read script")
		require.ErrorContains(t, err, "read kaboom")
	})

	t.Run("scriptReader.Close fails", func(t *testing.T) {
		t.Parallel()
		comp := createTestCompiler(t, "main")
		// Read succeeds (returns EOF on first Read with empty buffer is fine for
		// io.ReadAll, but we want non-empty bytes so Close error is the only failure).
		reader := &readCloseErr{
			buf:      bytes.NewReader([]byte("anything")),
			closeErr: errors.New("close kaboom"),
		}

		execContent, err := comp.Compile(t.Context(), reader)
		require.Error(t, err)
		require.Nil(t, execContent)
		require.ErrorContains(t, err, "failed to close reader")
		require.ErrorContains(t, err, "close kaboom")
	})

	t.Run("compileFn returns nil plugin without error", func(t *testing.T) {
		t.Parallel()
		comp := createTestCompiler(t, "main")
		comp.compileFn = func(
			ctx context.Context,
			wasmBytes []byte,
			opts *compile.Settings,
		) (adapters.CompiledPlugin, error) {
			return nil, nil
		}

		reader := newMockScriptReaderCloser([]byte("any-bytes"))
		reader.On("Close").Return(nil)

		execContent, err := comp.Compile(t.Context(), reader)
		require.Error(t, err)
		require.Nil(t, execContent)
		require.ErrorIs(t, err, ErrBytecodeNil)
		reader.AssertExpectations(t)
	})

	t.Run("plugin.Instance fails", func(t *testing.T) {
		t.Parallel()
		comp := createTestCompiler(t, "main")

		plugin := &mockCompiledPlugin{}
		plugin.On("Instance", mock.Anything, mock.Anything).
			Return(nil, errors.New("instance kaboom"))

		comp.compileFn = func(
			ctx context.Context,
			wasmBytes []byte,
			opts *compile.Settings,
		) (adapters.CompiledPlugin, error) {
			return plugin, nil
		}

		reader := newMockScriptReaderCloser([]byte("any-bytes"))
		reader.On("Close").Return(nil)

		execContent, err := comp.Compile(t.Context(), reader)
		require.Error(t, err)
		require.Nil(t, execContent)
		require.ErrorIs(t, err, ErrValidationFailed)
		require.ErrorContains(t, err, "failed to create test instance")
		require.ErrorContains(t, err, "instance kaboom")
		plugin.AssertExpectations(t)
		reader.AssertExpectations(t)
	})

	t.Run("instance.Close error is logged, not returned", func(t *testing.T) {
		t.Parallel()
		comp := createTestCompiler(t, "main")

		instance := &mockPluginInstance{}
		instance.On("FunctionExists", "main").Return(true)
		instance.On("Close", mock.Anything).Return(errors.New("close kaboom"))

		plugin := &mockCompiledPlugin{}
		plugin.On("Instance", mock.Anything, mock.Anything).Return(instance, nil)

		comp.compileFn = func(
			ctx context.Context,
			wasmBytes []byte,
			opts *compile.Settings,
		) (adapters.CompiledPlugin, error) {
			return plugin, nil
		}

		reader := newMockScriptReaderCloser([]byte("any-bytes"))
		reader.On("Close").Return(nil)

		execContent, err := comp.Compile(t.Context(), reader)
		require.NoError(t, err, "instance.Close error must not propagate")
		require.NotNil(t, execContent)
		plugin.AssertExpectations(t)
		instance.AssertExpectations(t)
		reader.AssertExpectations(t)
	})

	t.Run("FunctionExists returns false", func(t *testing.T) {
		t.Parallel()
		comp := createTestCompiler(t, "no_such_fn")

		instance := &mockPluginInstance{}
		instance.On("FunctionExists", "no_such_fn").Return(false)
		instance.On("Close", mock.Anything).Return(nil)

		plugin := &mockCompiledPlugin{}
		plugin.On("Instance", mock.Anything, mock.Anything).Return(instance, nil)

		comp.compileFn = func(
			ctx context.Context,
			wasmBytes []byte,
			opts *compile.Settings,
		) (adapters.CompiledPlugin, error) {
			return plugin, nil
		}

		reader := newMockScriptReaderCloser([]byte("any-bytes"))
		reader.On("Close").Return(nil)

		execContent, err := comp.Compile(t.Context(), reader)
		require.Error(t, err)
		require.Nil(t, execContent)
		require.ErrorIs(t, err, ErrValidationFailed)
		require.ErrorContains(t, err, "entry point function 'no_such_fn' not found")
		plugin.AssertExpectations(t)
		instance.AssertExpectations(t)
		reader.AssertExpectations(t)
	})
}

// readCloseErr is an io.ReadCloser that delegates Read to a *bytes.Reader and
// returns a configured error from Close. Used to exercise the scriptReader.Close
// failure branch without involving the testify mock machinery.
type readCloseErr struct {
	buf      *bytes.Reader
	closeErr error
}

func (r *readCloseErr) Read(p []byte) (int, error) {
	return r.buf.Read(p)
}

func (r *readCloseErr) Close() error {
	return r.closeErr
}
