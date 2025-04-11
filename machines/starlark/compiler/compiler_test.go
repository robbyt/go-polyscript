package compiler

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockScriptReaderCloser implements io.ReadCloser for testing
type mockScriptReaderCloser struct {
	*mock.Mock
	content string
	offset  int
}

func newMockScriptReaderCloser(content string) *mockScriptReaderCloser {
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

// mockErrorReader implements io.ReadCloser for testing read errors
type mockErrorReader struct{}

func (m *mockErrorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func (m *mockErrorReader) Close() error {
	return nil
}

func TestNewCompiler(t *testing.T) {
	t.Parallel()

	t.Run("basic creation", func(t *testing.T) {
		comp, err := New(
			WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)
		require.Equal(t, "starlark.Compiler", comp.String())
	})

	t.Run("with globals", func(t *testing.T) {
		globals := []string{"request", "response"}
		comp, err := New(
			WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
			WithGlobals(globals),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)
	})

	t.Run("with ctx global", func(t *testing.T) {
		comp, err := New(
			WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
			WithCtxGlobal(),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)
	})

	t.Run("defaults", func(t *testing.T) {
		comp, err := New()
		require.NoError(t, err)
		require.NotNil(t, comp)
	})
}

func TestCompiler_Compile(t *testing.T) {
	t.Parallel()

	t.Run("success cases", func(t *testing.T) {
		successTests := []struct {
			name    string
			script  string
			globals []string
		}{
			{
				name:    "valid script",
				script:  `print("Hello, World!")`,
				globals: []string{"request"},
			},
			{
				name:    "only comments",
				script:  `# This is just a comment`,
				globals: []string{"request"},
			},
			{
				name:    "with multiple globals",
				script:  `print(request, response)`,
				globals: []string{"request", "response"},
			},
			{
				name: "complex valid script with global override",
				script: `
request = True
def main():
    if request:
        print("Yes")
    else:
        print("No")
main()
`,
				globals: []string{"request"},
			},
			{
				name: "complex valid script with condition",
				script: `
def main():
    if condition:
        print("Yes")
    else:
        print("No")
main()
`,
				globals: []string{"condition"},
			},
		}

		for _, tt := range successTests {
			t.Run(tt.name, func(t *testing.T) {
				// Create compiler with options
				comp, err := New(
					WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
					WithGlobals(tt.globals),
				)
				require.NoError(t, err, "Failed to create compiler")

				reader := io.ReadCloser(newMockScriptReaderCloser(tt.script))
				if mockReader, ok := reader.(*mockScriptReaderCloser); ok {
					mockReader.On("Close").Return(nil)
				} else {
					t.Fatal("Failed to create mock reader")
				}

				// Execute test
				execContent, err := comp.Compile(reader)
				require.NoError(t, err, "Did not expect an error but got one")
				require.NotNil(t, execContent, "Expected execContent to be non-nil")
				require.Equal(
					t,
					tt.script,
					execContent.GetSource(),
					"Script content does not match",
				)

				// Verify mock expectations
				if mockReader, ok := reader.(*mockScriptReaderCloser); ok {
					mockReader.AssertExpectations(t)
				}
			})
		}
	})

	t.Run("error cases", func(t *testing.T) {
		errorTests := []struct {
			name    string
			script  string
			globals []string
			err     error
		}{
			{
				name:    "syntax error - missing closing parenthesis",
				script:  `print("Hello, World!"`,
				globals: []string{"request"},
				err:     ErrValidationFailed,
			},
			{
				name:    "empty script",
				script:  ``,
				globals: []string{"request"},
				err:     ErrContentNil,
			},
			{
				name:    "undefined global",
				script:  `print(undefined_global)`,
				globals: []string{"request"},
				err:     ErrValidationFailed,
			},
			{
				name:    "script using undefined global",
				script:  `print(undefined)`,
				globals: []string{"request"},
				err:     ErrValidationFailed,
			},
		}

		for _, tt := range errorTests {
			t.Run(tt.name, func(t *testing.T) {
				// Create compiler with options
				comp, err := New(
					WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
					WithGlobals(tt.globals),
				)
				require.NoError(t, err, "Failed to create compiler")

				reader := io.ReadCloser(newMockScriptReaderCloser(tt.script))
				if mockReader, ok := reader.(*mockScriptReaderCloser); ok {
					mockReader.On("Close").Return(nil)
				} else {
					t.Fatal("Failed to create mock reader")
				}

				// Execute test
				execContent, err := comp.Compile(reader)
				require.Error(t, err, "Expected an error but got none")
				require.Nil(t, execContent, "Expected execContent to be nil")
				require.True(t, errors.Is(err, tt.err), "Expected error %v, got %v", tt.err, err)

				// Verify mock expectations
				if mockReader, ok := reader.(*mockScriptReaderCloser); ok {
					mockReader.AssertExpectations(t)
				}
			})
		}

		t.Run("nil reader", func(t *testing.T) {
			comp, err := New(WithLogHandler(slog.NewTextHandler(os.Stdout, nil)))
			require.NoError(t, err)
			require.NotNil(t, comp, "Expected compiler to be non-nil")

			execContent, err := comp.Compile(nil)
			require.Error(t, err, "Expected an error but got none")
			require.Nil(t, execContent, "Expected execContent to be nil")
			require.True(t, errors.Is(err, ErrContentNil), "Expected error to be ErrContentNil")
		})

		t.Run("io error", func(t *testing.T) {
			comp, err := New(
				WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
				WithGlobals([]string{"ctx"}),
			)
			require.NoError(t, err)
			require.NotNil(t, comp, "Expected compiler to be non-nil")

			// Create a reader that will return an error
			reader := &mockErrorReader{}
			execContent, err := comp.Compile(reader)
			require.Error(t, err, "Expected an error but got none")
			require.Nil(t, execContent, "Expected execContent to be nil")
			require.Contains(
				t,
				err.Error(),
				"failed to read script",
				"Expected error to contain 'failed to read script'",
			)
		})

		t.Run("close error", func(t *testing.T) {
			comp, err := New(
				WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
			)
			require.NoError(t, err)
			require.NotNil(t, comp, "Expected compiler to be non-nil")

			// Create a reader that will return an error on close
			reader := newMockScriptReaderCloser(`print("Hello, World!")`)
			reader.On("Close").Return(errors.New("test error")).Once()

			execContent, err := comp.Compile(reader)
			require.Error(t, err, "Expected an error but got none")
			require.Nil(t, execContent, "Expected execContent to be nil")
			require.Contains(
				t,
				err.Error(),
				"failed to close reader",
				"Expected error to contain 'failed to close reader'",
			)
		})
	})
}

func TestCompilerOptions(t *testing.T) {
	t.Parallel()

	t.Run("WithLogHandler option", func(t *testing.T) {
		// Create a custom handler
		handler := slog.NewTextHandler(os.Stdout, nil)
		comp, err := New(WithLogHandler(handler))
		require.NoError(t, err)
		require.NotNil(t, comp)
		require.Equal(t, "starlark.Compiler", comp.String())
	})

	t.Run("WithLogger option", func(t *testing.T) {
		// Create a custom logger
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, nil)
		logger := slog.New(handler)
		comp, err := New(WithLogger(logger))
		require.NoError(t, err)
		require.NotNil(t, comp)
		require.Equal(t, "starlark.Compiler", comp.String())
	})

	t.Run("WithGlobals option", func(t *testing.T) {
		globals := []string{"request", "response"}
		comp, err := New(
			WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
			WithGlobals(globals),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)

		// Test with a script using the globals
		script := `print(request, response)`
		reader := io.ReadCloser(newMockScriptReaderCloser(script))
		if mockReader, ok := reader.(*mockScriptReaderCloser); ok {
			mockReader.On("Close").Return(nil)
		}

		execContent, err := comp.Compile(reader)
		require.NoError(t, err)
		require.NotNil(t, execContent)
	})

	t.Run("WithCtxGlobal option", func(t *testing.T) {
		comp, err := New(
			WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
			WithCtxGlobal(),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)

		// Test with a script using the ctx global
		script := `print(ctx)`
		reader := io.ReadCloser(newMockScriptReaderCloser(script))
		if mockReader, ok := reader.(*mockScriptReaderCloser); ok {
			mockReader.On("Close").Return(nil)
		}

		execContent, err := comp.Compile(reader)
		require.NoError(t, err)
		require.NotNil(t, execContent)
	})

	t.Run("Default options", func(t *testing.T) {
		// Test with no explicit options
		comp, err := New()
		require.NoError(t, err)
		require.NotNil(t, comp)

		// Simple script that doesn't require globals
		script := `print("Hello")`
		reader := io.ReadCloser(newMockScriptReaderCloser(script))
		if mockReader, ok := reader.(*mockScriptReaderCloser); ok {
			mockReader.On("Close").Return(nil)
		}

		execContent, err := comp.Compile(reader)
		require.NoError(t, err)
		require.NotNil(t, execContent)
	})

	t.Run("Option error handling", func(t *testing.T) {
		// Test with nil logger
		_, err := New(WithLogger(nil))
		require.Error(t, err)
		require.Contains(t, err.Error(), "logger cannot be nil")

		// Test with nil handler
		_, err = New(WithLogHandler(nil))
		require.Error(t, err)
		require.Contains(t, err.Error(), "log handler cannot be nil")
	})
}

func TestCompileError(t *testing.T) {
	// Test that the compiler returns the correct error when the script is nil
	comp, err := New(WithLogHandler(slog.NewTextHandler(os.Stdout, nil)))
	require.NoError(t, err)
	require.NotNil(t, comp, "Expected compiler to be non-nil")

	// Execute test with nil reader
	execContent, err := comp.Compile(nil)
	require.Error(t, err, "Expected an error but got none")
	require.Nil(t, execContent, "Expected execContent to be nil")
	require.True(t, errors.Is(err, ErrContentNil), "Expected error to be ErrContentNil")
}

func TestCompileIOError(t *testing.T) {
	// Test that we return the correct error when there's an IO error
	comp, err := New(
		WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
		WithGlobals([]string{"ctx"}),
	)
	require.NoError(t, err)
	require.NotNil(t, comp, "Expected compiler to be non-nil")

	// Create a reader that will return an error
	reader := &mockErrorReader{}
	execContent, err := comp.Compile(reader)
	require.Error(t, err, "Expected an error but got none")
	require.Nil(t, execContent, "Expected execContent to be nil")
	require.Contains(
		t,
		err.Error(),
		"failed to read script",
		"Expected error to contain 'failed to read script'",
	)
}

func TestCompileCloseError(t *testing.T) {
	// Test that we return the correct error when there's an error closing the reader
	comp, err := New(
		WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
	)
	require.NoError(t, err)
	require.NotNil(t, comp, "Expected compiler to be non-nil")

	// Create a reader that will return an error on close
	reader := newMockScriptReaderCloser(`print("Hello, World!")`)
	reader.On("Close").Return(errors.New("test error")).Once()

	execContent, err := comp.Compile(reader)
	require.Error(t, err, "Expected an error but got none")
	require.Nil(t, execContent, "Expected execContent to be nil")
	require.Contains(
		t,
		err.Error(),
		"failed to close reader",
		"Expected error to contain 'failed to close reader'",
	)
}

func TestCompilerString(t *testing.T) {
	comp, err := New(WithLogHandler(slog.NewTextHandler(os.Stdout, nil)))
	require.NoError(t, err)
	require.NotNil(t, comp, "Expected compiler to be non-nil")
	require.Equal(
		t,
		"starlark.Compiler",
		comp.String(),
		"Expected compiler name to be starlark.Compiler",
	)
}
