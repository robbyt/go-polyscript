package script

import (
	"errors"
	"io"
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

// TestCompiler tests the Compile method of the Compiler interface.
func TestCompiler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string
		mockReturn  ExecutableContent
		mockError   error
		expectError bool
	}{
		{
			name:        "ValidScript",
			content:     "print('Hello, World!')",
			mockReturn:  &MockExecutableContent{},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "InvalidScript",
			content:     "invalid script content",
			mockReturn:  nil,
			mockError:   errors.New("validation failed"),
			expectError: true,
		},
		{
			name:        "EmptyScript",
			content:     "",
			mockReturn:  nil,
			mockError:   errors.New("content is empty"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock compiler and reader
			mockCompiler := new(MockCompiler)
			reader := newMockScriptReaderCloser(tt.content)
			reader.On("Close").Return(nil).Maybe()

			// Set expectations
			mockCompiler.On("Compile", reader).Return(tt.mockReturn, tt.mockError)

			// Execute test
			result, err := mockCompiler.Compile(reader)

			// Verify results
			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}

			// Verify mocks
			mockCompiler.AssertExpectations(t)
			reader.AssertExpectations(t)
		})
	}
}
