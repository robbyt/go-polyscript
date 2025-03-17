package risor

import (
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
	args := m.Mock.Called()
	return args.Error(0)
}

func TestCompiler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		script  string
		globals []string
		err     error
	}{
		{
			name:    "valid script",
			script:  `print("Hello, World!")`,
			globals: []string{"request"},
		},
		{
			name:    "empty script",
			script:  "",
			globals: []string{"request"},
			err:     ErrContentNil,
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
			name:    "only comments",
			script:  `# This is just a comment`,
			globals: []string{"request"},
			err:     ErrNoInstructions,
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
			name: "complex valid script with top-scope override",
			script: `
request = true
func main() {
    if request {
        return "Yes"
    } else {
        return "No"
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
        return "Yes"
    } else {
        return "No"
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
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mock compiler and reader
			handler := slog.NewTextHandler(os.Stdout, nil)
			comp := NewCompiler(handler, &BasicCompilerOptions{Globals: tt.globals})
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

			// Verify mock expectations
			if mockReader, ok := reader.(*mockScriptReaderCloser); ok {
				mockReader.AssertExpectations(t)
			}
		})
	}
}

// TestValidateAdditionalCases also needs to be updated similarly
func TestValidateAdditionalCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		script      string
		globals     []string
		expectError bool
		err         error
	}{
		{
			name:        "valid script with globals",
			script:      `print(request)`,
			globals:     []string{"request"},
			expectError: false,
		},
		{
			name:        "valid script with multiple globals",
			script:      `print(request, response)`,
			globals:     []string{"request", "response"},
			expectError: false,
		},
		{
			name:        "script using undefined global",
			script:      `print(undefined)`,
			globals:     []string{"request"},
			expectError: true,
			err:         ErrValidationFailed,
		},
		{
			name:        "complex valid script with condition",
			script:      `if true { print("Yes") } else { print("No") }`,
			globals:     []string{"condition"},
			expectError: false,
		},
		{
			name:        "syntax error - missing closing parenthesis",
			script:      `print("Hello, World!"`,
			globals:     []string{"request"},
			expectError: true,
			err:         ErrValidationFailed,
		},
		{
			name:        "empty script",
			script:      ``,
			globals:     []string{"request"},
			expectError: true,
			err:         ErrContentNil,
		},
		{
			name:        "only comments",
			script:      `# This is just a comment`,
			globals:     []string{"request"},
			expectError: true,
			err:         ErrNoInstructions,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create compiler and reader
			handler := slog.NewTextHandler(os.Stdout, nil)
			comp := NewCompiler(handler, &BasicCompilerOptions{Globals: tt.globals})
			reader := io.ReadCloser(newMockScriptReaderCloser(tt.script))
			if mockReader, ok := reader.(*mockScriptReaderCloser); ok {
				mockReader.On("Close").Return(nil)
			} else {
				t.Fatal("Failed to create mock reader")
			}

			// Execute test
			execContent, err := comp.Compile(reader)

			if tt.expectError {
				require.Error(t, err, "Expected an error but got none")
				require.Nil(t, execContent, "Expected execContent to be nil")
				require.True(t, errors.Is(err, tt.err))
			} else {
				require.NoError(t, err, "Did not expect an error but got one")
				require.NotNil(t, execContent, "Expected execContent to be non-nil")
				require.Equal(t, tt.script, execContent.GetSource(), "Script content does not match")
			}

			// Verify mock expectations
			if mockReader, ok := reader.(*mockScriptReaderCloser); ok {
				mockReader.AssertExpectations(t)
			}
		})
	}
}
