package loader

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
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
			expectedType  any
			shouldError   bool
			errorContains string
		}{
			{
				name:         "HTTP URL",
				input:        "http://example.com/script.js",
				expectedType: (*FromHTTP)(nil),
			},
			{
				name:         "HTTPS URL",
				input:        "https://example.com/script.js",
				expectedType: (*FromHTTP)(nil),
			},
			{
				name:         "file URL",
				input:        "file:///path/to/script.js",
				expectedType: (*FromDisk)(nil),
			},
			{
				name:         "absolute path",
				input:        "/absolute/path/script.js",
				expectedType: (*FromDisk)(nil),
			},
			{
				name:         "inline script content",
				input:        "function test() { return 'hello'; }",
				expectedType: (*FromString)(nil),
			},
			{
				name:         "multiline script content",
				input:        "function test() {\n  return 'hello';\n}",
				expectedType: (*FromString)(nil),
			},
			{
				name: "base64 encoded content",
				input: base64.StdEncoding.EncodeToString(
					[]byte("console.log('base64 test');"),
				),
				expectedType: (*FromBytes)(nil),
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
				assert.IsType(t, tc.expectedType, result, "Type mismatch for input: %s", tc.input)
			})
		}
	})

	t.Run("byte slice inputs", func(t *testing.T) {
		tests := []struct {
			name         string
			input        []byte
			expectedType any
			shouldError  bool
		}{
			{
				name:         "non-empty bytes",
				input:        []byte("function test() { return 'hello'; }"),
				expectedType: (*FromBytes)(nil),
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
				assert.IsType(t, tc.expectedType, result, "Type mismatch for input")
			})
		}
	})

	t.Run("io.Reader inputs", func(t *testing.T) {
		content := "function test() { return 'hello'; }"
		reader := strings.NewReader(content)

		result, err := InferLoader(reader)

		require.NoError(t, err)
		assert.IsType(t, (*FromIoReader)(nil), result)
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
			expectedType any
		}{
			{
				name:         "http scheme",
				input:        "http://localhost:8080/script.js",
				expectedType: (*FromHTTP)(nil),
			},
			{
				name:         "https scheme",
				input:        "https://api.example.com/script.js",
				expectedType: (*FromHTTP)(nil),
			},
			{
				name:         "file scheme",
				input:        "file:///usr/local/scripts/test.js",
				expectedType: (*FromDisk)(nil),
			},
			{
				name:         "custom scheme treated as content",
				input:        "custom://path/to/script",
				expectedType: (*FromString)(nil),
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := inferFromString(tc.input)

				require.NoError(t, err, "Unexpected error for input: %s", tc.input)
				assert.IsType(t, tc.expectedType, result, "Type mismatch for input: %s", tc.input)
			})
		}
	})

	t.Run("path detection", func(t *testing.T) {
		tests := []struct {
			name         string
			input        string
			expectedType any
		}{
			{
				name:         "absolute unix path",
				input:        "/usr/local/bin/script.js",
				expectedType: (*FromDisk)(nil),
			},
			{
				name:         "path with forward slash",
				input:        "some/path/script.js",
				expectedType: (*FromDisk)(nil),
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := inferFromString(tc.input)

				require.NoError(t, err, "Unexpected error for input: %s", tc.input)
				assert.IsType(t, tc.expectedType, result)
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
			expectedType any
		}{
			{
				name:         "windows absolute path",
				input:        "C:\\Program Files\\script.js",
				expectedType: (*FromDisk)(nil),
			},
			{
				name:         "windows drive with colon",
				input:        "D:\\scripts\\test.js",
				expectedType: (*FromDisk)(nil),
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := inferFromString(tc.input)

				require.NoError(t, err, "Unexpected error for input: %s", tc.input)
				assert.IsType(t, tc.expectedType, result)
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
				assert.IsType(t, (*FromString)(nil), result)
			})
		}
	})

	t.Run("base64 handling", func(t *testing.T) {
		tests := []struct {
			name            string
			input           string
			expectedType    any
			expectedContent string
		}{
			{
				name:            "valid base64 script",
				input:           base64.StdEncoding.EncodeToString([]byte("console.log('hello');")),
				expectedType:    (*FromBytes)(nil),
				expectedContent: "console.log('hello');",
			},
			{
				name: "valid base64 multiline",
				input: base64.StdEncoding.EncodeToString(
					[]byte("function test() {\n  return 42;\n}"),
				),
				expectedType:    (*FromBytes)(nil),
				expectedContent: "function test() {\n  return 42;\n}",
			},
			{
				name:            "invalid base64 falls back to string",
				input:           "not-base64-content",
				expectedType:    (*FromString)(nil),
				expectedContent: "not-base64-content",
			},
			{
				name:            "plain text that looks like base64",
				input:           base64.StdEncoding.EncodeToString([]byte("Hello World")),
				expectedType:    (*FromBytes)(nil),
				expectedContent: "Hello World",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := InferLoader(tc.input)
				require.NoError(t, err)

				// Check loader type
				assert.IsType(t, tc.expectedType, result)

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
	assert.IsType(t, (*FromDisk)(nil), result)
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

	t.Run("file path inference with temp file", func(t *testing.T) {
		// Create a temporary directory and file
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "test_script.js")

		// Write some content to the file
		content := "console.log('temporary file test');"
		err := os.WriteFile(tempFile, []byte(content), 0o644)
		require.NoError(t, err)

		// Test that InferLoader returns a FromDisk loader for the file path
		result, err := InferLoader(tempFile)
		require.NoError(t, err)
		assert.IsType(t, (*FromDisk)(nil), result)

		// Verify the content can be read
		reader, err := result.GetReader()
		require.NoError(t, err)
		t.Cleanup(func() {
			assert.NoError(t, reader.Close())
		})

		actualContent, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, content, string(actualContent))
	})
}
