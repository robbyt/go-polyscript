package loader

import (
	"encoding/base64"
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInferLoader(t *testing.T) {
	t.Parallel()

	t.Run("string inputs", func(t *testing.T) {
		tests := []struct {
			name          string
			input         string
			expectedType  string
			shouldError   bool
			errorContains string
		}{
			{
				name:         "HTTP URL",
				input:        "http://example.com/script.js",
				expectedType: "*loader.FromHTTP",
			},
			{
				name:         "HTTPS URL",
				input:        "https://example.com/script.js",
				expectedType: "*loader.FromHTTP",
			},
			{
				name:         "file URL",
				input:        "file:///path/to/script.js",
				expectedType: "*loader.FromDisk",
			},
			{
				name:         "absolute path",
				input:        "/absolute/path/script.js",
				expectedType: "*loader.FromDisk",
			},
			{
				name:         "inline script content",
				input:        "function test() { return 'hello'; }",
				expectedType: "*loader.FromString",
			},
			{
				name:         "multiline script content",
				input:        "function test() {\n  return 'hello';\n}",
				expectedType: "*loader.FromString",
			},
			{
				name:         "base64 encoded content",
				input:        base64.StdEncoding.EncodeToString([]byte("console.log('base64 test');")),
				expectedType: "*loader.FromBytes",
			},
			{
				name:        "empty string",
				input:       "",
				shouldError: true,
			},
			{
				name:        "whitespace only",
				input:       "   \n\t  ",
				shouldError: true,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := InferLoader(tc.input)

				if tc.shouldError {
					assert.Error(t, err, "Expected error for input: %s", tc.input)
					if tc.errorContains != "" {
						assert.Contains(t, err.Error(), tc.errorContains)
					}
					return
				}

				require.NoError(t, err, "Unexpected error for input: %s", tc.input)
				assert.Equal(
					t,
					tc.expectedType,
					getTypeName(result),
					"Type mismatch for input: %s",
					tc.input,
				)
			})
		}
	})

	t.Run("byte slice inputs", func(t *testing.T) {
		tests := []struct {
			name         string
			input        []byte
			expectedType string
			shouldError  bool
		}{
			{
				name:         "non-empty bytes",
				input:        []byte("function test() { return 'hello'; }"),
				expectedType: "*loader.FromBytes",
			},
			{
				name:        "empty bytes",
				input:       []byte{},
				shouldError: true,
			},
			{
				name:        "whitespace only bytes",
				input:       []byte("   \n\t  "),
				shouldError: true,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := InferLoader(tc.input)

				if tc.shouldError {
					assert.Error(t, err, "Expected error for input")
					return
				}

				require.NoError(t, err, "Unexpected error for input")
				assert.Equal(t, tc.expectedType, getTypeName(result), "Type mismatch for input")
			})
		}
	})

	t.Run("io.Reader inputs", func(t *testing.T) {
		content := "function test() { return 'hello'; }"
		reader := strings.NewReader(content)

		result, err := InferLoader(reader)

		require.NoError(t, err)
		assert.Equal(t, "*loader.FromIoReader", getTypeName(result))
	})

	t.Run("existing loader input", func(t *testing.T) {
		originalLoader, err := NewFromString("test content")
		require.NoError(t, err)

		result, err := InferLoader(originalLoader)

		require.NoError(t, err)
		assert.Same(t, originalLoader, result, "Should return the same loader instance")
	})

	t.Run("unsupported input types", func(t *testing.T) {
		tests := []struct {
			name  string
			input any
		}{
			{"int", 42},
			{"float", 3.14},
			{"bool", true},
			{"struct", struct{ Name string }{Name: "test"}},
			{"map", map[string]string{"key": "value"}},
			{"slice", []string{"item1", "item2"}},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := InferLoader(tc.input)

				assert.Error(t, err, "Expected error for unsupported type")
				assert.Nil(t, result, "Result should be nil for unsupported type")
				assert.Contains(t, err.Error(), "unsupported input type")
			})
		}
	})
}

func TestInferFromString(t *testing.T) {
	t.Parallel()

	t.Run("URL scheme detection", func(t *testing.T) {
		tests := []struct {
			name         string
			input        string
			expectedType string
		}{
			{
				name:         "http scheme",
				input:        "http://localhost:8080/script.js",
				expectedType: "*loader.FromHTTP",
			},
			{
				name:         "https scheme",
				input:        "https://api.example.com/script.js",
				expectedType: "*loader.FromHTTP",
			},
			{
				name:         "file scheme",
				input:        "file:///usr/local/scripts/test.js",
				expectedType: "*loader.FromDisk",
			},
			{
				name:         "custom scheme treated as content",
				input:        "custom://path/to/script",
				expectedType: "*loader.FromString",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := inferFromString(tc.input)

				require.NoError(t, err, "Unexpected error for input: %s", tc.input)
				assert.Equal(
					t,
					tc.expectedType,
					getTypeName(result),
					"Type mismatch for input: %s",
					tc.input,
				)
			})
		}
	})

	t.Run("path detection", func(t *testing.T) {
		tests := []struct {
			name         string
			input        string
			expectedType string
		}{
			{
				name:         "absolute unix path",
				input:        "/usr/local/bin/script.js",
				expectedType: "*loader.FromDisk",
			},
			{
				name:         "path with forward slash",
				input:        "some/path/script.js",
				expectedType: "*loader.FromDisk",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := inferFromString(tc.input)

				require.NoError(t, err, "Unexpected error for input: %s", tc.input)
				assert.Equal(
					t,
					tc.expectedType,
					getTypeName(result),
					"Type mismatch for input: %s",
					tc.input,
				)
			})
		}
	})

	t.Run("windows path detection", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("Windows path tests only run on Windows")
		}

		tests := []struct {
			name         string
			input        string
			expectedType string
		}{
			{
				name:         "windows absolute path",
				input:        "C:\\Program Files\\script.js",
				expectedType: "*loader.FromDisk",
			},
			{
				name:         "windows drive with colon",
				input:        "D:\\scripts\\test.js",
				expectedType: "*loader.FromDisk",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := inferFromString(tc.input)

				require.NoError(t, err, "Unexpected error for input: %s", tc.input)
				assert.Equal(
					t,
					tc.expectedType,
					getTypeName(result),
					"Type mismatch for input: %s",
					tc.input,
				)
			})
		}
	})

	t.Run("inline content detection", func(t *testing.T) {
		tests := []string{
			"function test() { return 'hello'; }",
			"var x = 42;",
			"console.log('hello world');",
			"return 1 + 2 + 3;",
			"simple script content",
		}

		for _, input := range tests {
			t.Run(input, func(t *testing.T) {
				result, err := inferFromString(input)

				require.NoError(t, err, "Unexpected error for input: %s", input)
				assert.Equal(
					t,
					"*loader.FromString",
					getTypeName(result),
					"Should detect as inline content",
				)
			})
		}
	})

	t.Run("base64 handling", func(t *testing.T) {
		tests := []struct {
			name            string
			input           string
			expectedType    string
			expectedContent string
		}{
			{
				name:            "valid base64 script",
				input:           base64.StdEncoding.EncodeToString([]byte("console.log('hello');")),
				expectedType:    "*loader.FromBytes",
				expectedContent: "console.log('hello');",
			},
			{
				name:            "valid base64 multiline",
				input:           base64.StdEncoding.EncodeToString([]byte("function test() {\n  return 42;\n}")),
				expectedType:    "*loader.FromBytes",
				expectedContent: "function test() {\n  return 42;\n}",
			},
			{
				name:            "invalid base64 falls back to string",
				input:           "not-base64-content",
				expectedType:    "*loader.FromString",
				expectedContent: "not-base64-content",
			},
			{
				name:            "plain text that looks like base64",
				input:           base64.StdEncoding.EncodeToString([]byte("Hello World")),
				expectedType:    "*loader.FromBytes",
				expectedContent: "Hello World",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := InferLoader(tc.input)
				require.NoError(t, err)

				// Check loader type
				assert.Equal(t, tc.expectedType, getTypeName(result))

				// Verify content
				reader, err := result.GetReader()
				require.NoError(t, err)
				defer func() {
					assert.NoError(t, reader.Close())
				}()

				content, err := io.ReadAll(reader)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedContent, string(content))
			})
		}
	})
}

func TestInferLoader_WindowsPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows path test only runs on Windows")
	}

	result, err := InferLoader("C:\\windows\\path\\script.js")
	require.NoError(t, err)
	assert.Equal(t, "*loader.FromDisk", getTypeName(result))
}

func TestInferLoader_Integration(t *testing.T) {
	t.Parallel()

	t.Run("can create reader from inferred loader", func(t *testing.T) {
		tests := []struct {
			name     string
			input    any
			expected string
		}{
			{
				name:     "string content",
				input:    "function test() { return 'hello'; }",
				expected: "function test() { return 'hello'; }",
			},
			{
				name:     "byte content",
				input:    []byte("return 42;"),
				expected: "return 42;",
			},
			{
				name:     "reader content",
				input:    strings.NewReader("console.log('test');"),
				expected: "console.log('test');",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				inferredLoader, err := InferLoader(tc.input)
				require.NoError(t, err)

				reader, err := inferredLoader.GetReader()
				require.NoError(t, err)

				defer func() {
					assert.NoError(t, reader.Close())
				}()

				content, err := io.ReadAll(reader)
				require.NoError(t, err)

				assert.Equal(t, tc.expected, string(content))
			})
		}
	})

	t.Run("source URL is properly set", func(t *testing.T) {
		tests := []struct {
			name        string
			input       any
			expectedURL string
		}{
			{
				name:        "string content",
				input:       "test content",
				expectedURL: "string://inline/",
			},
			{
				name:        "byte content",
				input:       []byte("test content"),
				expectedURL: "bytes://inline/",
			},
			{
				name:        "reader content",
				input:       strings.NewReader("test content"),
				expectedURL: "reader://inferred/",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				inferredLoader, err := InferLoader(tc.input)
				require.NoError(t, err)

				sourceURL := inferredLoader.GetSourceURL()
				assert.Contains(t, sourceURL.String(), tc.expectedURL)
			})
		}
	})
}

// Helper function to get type name for assertions
func getTypeName(v any) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("%T", v)
}
