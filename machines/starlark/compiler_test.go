package starlark

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
	args := m.Called()
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
		{
			name:    "script using undefined global",
			script:  `print(undefined)`,
			globals: []string{"request"},
			err:     ErrValidationFailed,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mock compiler and reader
			handler := slog.NewTextHandler(os.Stdout, nil)
			comp := NewCompiler(handler, &StarlarkOptions{Globals: tt.globals})
			reader := newMockScriptReaderCloser(tt.script)
			reader.On("Close").Return(nil)

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
			reader.AssertExpectations(t)
		})
	}
}
