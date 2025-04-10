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

type testCase struct {
	name    string
	script  string
	globals []string
	err     error
}

// execute a single unit test
func runTestCase(t *testing.T, tt testCase) {
	t.Helper()
	t.Parallel()

	// Create compiler with options
	comp, err := NewCompiler(
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

	if tt.err != nil {
		require.Error(t, err, "Expected an error but got none")
		require.Nil(t, execContent, "Expected execContent to be nil")
		require.True(t, errors.Is(err, tt.err), "Expected error %v, got %v", tt.err, err)
		return
	}

	require.NoError(t, err, "Did not expect an error but got one")
	require.NotNil(t, execContent, "Expected execContent to be non-nil")
	require.Equal(t, tt.script, execContent.GetSource(), "Script content does not match")

	// Check that the bytecode is correct
	risorExec, ok := execContent.(*executable)
	require.True(t, ok, "Expected execContent to be a *Executable")
	require.NotNil(t, risorExec.GetRisorByteCode(), "Expected bytecode to be non-nil")

	// Verify mock expectations
	if mockReader, ok := reader.(*mockScriptReaderCloser); ok {
		mockReader.AssertExpectations(t)
	}
}

func TestCompiler(t *testing.T) {
	t.Parallel()
	tests := []testCase{
		{
			name:    "valid script",
			script:  `print("Hello, World!")`,
			globals: []string{"request"},
		},
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
			name:    "with multiple globals",
			script:  `print(request, response)`,
			globals: []string{"request", "response"},
		},
		{
			name: "complex valid script with global override",
			script: `
request = true
func main() {
    if request {
        print("Yes")
    } else {
        print("No")
    }
}
main()
`,
			globals: []string{"request"},
		},
		{
			name: "complex valid script with condition",
			script: `
func main() {
    if condition {
        print("Yes")
    } else {
        print("No")
    }
}
main()
`,
			globals: []string{"condition"},
		},
		{
			name:    "script using undefined global",
			script:  `print(undefined)`,
			globals: []string{"request"},
			err:     ErrValidationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runTestCase(t, tt)
		})
	}
}

func TestCompilerOptions(t *testing.T) {
	t.Parallel()

	t.Run("WithLogHandler option", func(t *testing.T) {
		// Create a custom handler
		handler := slog.NewTextHandler(os.Stdout, nil)
		comp, err := NewCompiler(WithLogHandler(handler))
		require.NoError(t, err)
		require.NotNil(t, comp)
		require.Equal(t, "risor.Compiler", comp.String())
	})

	t.Run("WithLogger option", func(t *testing.T) {
		// Create a custom logger
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, nil)
		logger := slog.New(handler)
		comp, err := NewCompiler(WithLogger(logger))
		require.NoError(t, err)
		require.NotNil(t, comp)
		require.Equal(t, "risor.Compiler", comp.String())
	})

	t.Run("WithGlobals option", func(t *testing.T) {
		globals := []string{"request", "response"}
		comp, err := NewCompiler(
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

	t.Run("Default options", func(t *testing.T) {
		// Test with no explicit options
		comp, err := NewCompiler()
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
		_, err := NewCompiler(WithLogger(nil))
		require.Error(t, err)
		require.Contains(t, err.Error(), "logger cannot be nil")

		// Test with nil handler
		_, err = NewCompiler(WithLogHandler(nil))
		require.Error(t, err)
		require.Contains(t, err.Error(), "log handler cannot be nil")
	})
}

func TestCompileError(t *testing.T) {
	// Test that the compiler returns the correct error when the script is nil
	comp, err := NewCompiler(WithLogHandler(slog.NewTextHandler(os.Stdout, nil)))
	require.NoError(t, err)
	require.NotNil(t, comp, "Expected compiler to be non-nil")

	// Execute test with nil reader
	execContent, err := comp.Compile(nil)
	require.Error(t, err, "Expected an error but got none")
	require.Nil(t, execContent, "Expected execContent to be nil")
	require.True(t, errors.Is(err, ErrContentNil), "Expected error to be ErrContentNil")
}

func TestCompileWithBytecode(t *testing.T) {
	comp, err := NewCompiler(
		WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
		WithGlobals([]string{"ctx"}),
	)
	require.NoError(t, err)
	require.NotNil(t, comp, "Expected compiler to be non-nil")

	// Here we test that we can directly call the compile method with a byteslice
	scriptBytes := []byte(`print("Hello, World!")`)
	executable, err := comp.compile(scriptBytes)
	require.NoError(t, err, "Did not expect an error but got one")
	require.NotNil(t, executable, "Expected execContent to be non-nil")
	require.Equal(t, string(scriptBytes), executable.GetSource(), "Script content does not match")

	// Check that the bytecode is valid
	require.NotNil(t, executable.GetRisorByteCode(), "Expected bytecode to be non-nil")

	// Simple test to make sure the bytecode exists
	code := executable.GetRisorByteCode()
	require.NotNil(t, code, "Expected code to be non-nil")
}

func TestCompileIOError(t *testing.T) {
	// Test that we return the correct error when there's an IO error
	comp, err := NewCompiler(
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
	comp, err := NewCompiler(
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

// mockErrorReader implements io.ReadCloser for testing read errors
type mockErrorReader struct{}

func (m *mockErrorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func (m *mockErrorReader) Close() error {
	return nil
}

// Test compiler string representation
func TestCompilerString(t *testing.T) {
	comp, err := NewCompiler(WithLogHandler(slog.NewTextHandler(os.Stdout, nil)))
	require.NoError(t, err)
	require.NotNil(t, comp, "Expected compiler to be non-nil")
	require.Equal(t, "risor.Compiler", comp.String(), "Expected compiler name to be risor.Compiler")
}
