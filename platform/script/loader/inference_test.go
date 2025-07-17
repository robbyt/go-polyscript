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
				input:        "http://example.com/script.wasm",
				expectedType: (*FromHTTP)(nil),
			},
			{
				name:         "HTTPS URL",
				input:        "https://example.com/script.risor",
				expectedType: (*FromHTTP)(nil),
			},
			{
				name:         "file URL",
				input:        "file:///path/to/script.star",
				expectedType: (*FromDisk)(nil),
			},
			{
				name:  "absolute path with invalid extension",
				input: "/absolute/path/script.invalid",
				expectedType: (*FromDisk)(
					nil,
				), // Absolute paths are treated as file paths regardless of extension
			},
			{
				name:         "absolute path with wasm extension",
				input:        "/absolute/path/script.wasm",
				expectedType: (*FromDisk)(nil),
			},
			{
				name:         "absolute path with risor extension",
				input:        "/absolute/path/script.risor",
				expectedType: (*FromDisk)(nil),
			},
			{
				name:         "absolute path with star extension",
				input:        "/absolute/path/script.star",
				expectedType: (*FromDisk)(nil),
			},
			{
				name:         "absolute path with starlark extension",
				input:        "/absolute/path/script.starlark",
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
				input:        "http://localhost:8080/script.wasm",
				expectedType: (*FromHTTP)(nil),
			},
			{
				name:         "https scheme",
				input:        "https://api.example.com/script.risor",
				expectedType: (*FromHTTP)(nil),
			},
			{
				name:         "file scheme",
				input:        "file:///usr/local/scripts/test.star",
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
				name:  "absolute unix path with invalid extension",
				input: "/usr/local/bin/script.invalid",
				expectedType: (*FromDisk)(
					nil,
				), // Absolute paths are file paths even with unsupported extensions
			},
			{
				name:         "absolute unix path with wasm extension",
				input:        "/usr/local/bin/script.wasm",
				expectedType: (*FromDisk)(nil),
			},
			{
				name:         "path with forward slash and invalid extension",
				input:        "some/path/script.invalid",
				expectedType: (*FromString)(nil),
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
				name:         "windows absolute path with invalid extension",
				input:        "C:\\Program Files\\script.invalid",
				expectedType: (*FromString)(nil),
			},
			{
				name:         "windows drive with colon and invalid extension",
				input:        "D:\\scripts\\test.invalid",
				expectedType: (*FromString)(nil),
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

	result, err := InferLoader("C:\\windows\\path\\script.starlark")
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
		tempFile := filepath.Join(tempDir, "test_script.wasm")

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

// TestInferLoader_MultilineScriptContent tests multiline content that might be misinterpreted as paths.
func TestInferLoader_MultilineScriptContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name: "script with comment header",
			input: `// JavaScript comment
function test() {
    return "hello world";
}`,
		},
		{
			name: "script starting with path-like comment",
			input: `// /usr/bin/node
const fs = require('fs');
console.log("test");`,
		},
		{
			name: "script with windows path in content",
			input: `const path = "C:\\Program Files\\App\\script.invalid";
function loadScript() {
    return path;
}`,
		},
		{
			name:  "multiline with control characters",
			input: "function test() {\n\treturn 'hello\\nworld';\n}",
		},
		{
			name: "content starting with /usr/bin",
			input: `/usr/bin/node
function test() { return 42; }`,
		},
		{
			name: "risor script with data access patterns",
			input: `func process() {
    service_name := ctx.get("service_name", "unknown")
    version := ctx.get("version", "1.0.0") 
    return {"message": "Hello from Risor!", "version": version}
}
process()`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := InferLoader(tc.input)
			require.NoError(t, err, "Input: %s", tc.input)
			assert.IsType(
				t,
				(*FromString)(nil),
				result,
				"Should treat multiline content as string, not path",
			)

			// Verify content can be read correctly
			reader, err := result.GetReader()
			require.NoError(t, err)
			defer func() {
				assert.NoError(t, reader.Close())
			}()

			content, err := io.ReadAll(reader)
			require.NoError(t, err)
			assert.Equal(t, tc.input, string(content))
		})
	}
}

// TestInferLoader_AmbiguousContentDetection tests edge cases between paths and content.
func TestInferLoader_AmbiguousContentDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		expectedType any
		description  string
	}{
		{
			name:         "path that doesn't exist should return FromString",
			input:        "nonexistent/path/script.invalid",
			expectedType: (*FromString)(nil),
			description:  "Paths with unsupported extensions should be treated as content",
		},
		{
			name:         "path with spaces and unsupported extension",
			input:        "some path with spaces.invalid",
			expectedType: (*FromString)(nil),
			description:  "Paths with unsupported extensions should be treated as content",
		},
		{
			name:         "relative path without extension",
			input:        "relative/path/file",
			expectedType: (*FromString)(nil),
			description:  "Relative paths should be treated as content",
		},
		{
			name:         "wasm file extension",
			input:        "script.wasm",
			expectedType: (*FromDisk)(nil),
			description:  "WASM files should be treated as paths",
		},
		{
			name:         "risor file extension",
			input:        "script.risor",
			expectedType: (*FromDisk)(nil),
			description:  "Risor files should be treated as paths",
		},
		{
			name:         "star file extension",
			input:        "config.star",
			expectedType: (*FromDisk)(nil),
			description:  "Star files should be treated as paths",
		},
		{
			name:         "starlark file extension",
			input:        "build.starlark",
			expectedType: (*FromDisk)(nil),
			description:  "Starlark files should be treated as paths",
		},
		{
			name:         "case insensitive WASM",
			input:        "MODULE.WASM",
			expectedType: (*FromDisk)(nil),
			description:  "Case insensitive file extensions should work",
		},
		{
			name:  "single line code that looks like path",
			input: "/bin/bash -c 'echo hello'",
			expectedType: (*FromDisk)(
				nil,
			), // Absolute paths are treated as file paths even if they look like commands
			description: "Absolute paths take precedence over content detection",
		},
		{
			name:         "function call looks like content",
			input:        "function() { return 42; }",
			expectedType: (*FromString)(nil),
			description:  "Function definitions should be treated as content",
		},
		{
			name:         "javascript var declaration",
			input:        "var x = '/some/path'; console.log(x);",
			expectedType: (*FromString)(nil),
			description:  "Variable declarations should be treated as content",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := InferLoader(tc.input)
			require.NoError(t, err)
			assert.IsType(t, tc.expectedType, result, tc.description)
		})
	}
}

// TestInferLoader_URLParsingEdgeCases tests graceful handling of URL parsing failures.
func TestInferLoader_URLParsingEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		shouldError   bool
		errorContains string
		expectedType  any
	}{
		{
			name:         "URL with control characters should fall back to content",
			input:        "http://example.com/script\nwith\nnewlines.invalid",
			shouldError:  false,
			expectedType: (*FromString)(nil),
		},
		{
			name:         "file URL with spaces should be handled gracefully",
			input:        "file:///path with spaces/script.wasm",
			shouldError:  false,
			expectedType: (*FromDisk)(nil), // Valid file URL should return FromDisk
		},
		{
			name:         "malformed scheme treated as content",
			input:        "ht!tp://invalid-scheme.com/script.invalid",
			expectedType: (*FromString)(nil),
		},
		{
			name:         "content that looks like URL with control chars",
			input:        "http://example.com\nfunction test() {}",
			expectedType: (*FromString)(nil),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := InferLoader(tc.input)

			if tc.shouldError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			require.NoError(t, err)
			if tc.expectedType != nil {
				assert.IsType(t, tc.expectedType, result)
			}
		})
	}
}

// TestInferLoader_ControlCharactersAndSpecialCases tests handling of various special characters.
func TestInferLoader_ControlCharactersAndSpecialCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		expectedType any
	}{
		{
			name:         "content with newlines and tabs",
			input:        "function test() {\n\treturn 'hello\\nworld';\n}",
			expectedType: (*FromString)(nil),
		},
		{
			name:         "content with carriage returns",
			input:        "line1\r\nline2\r\nfunction test() {}",
			expectedType: (*FromString)(nil),
		},
		{
			name: "content with null bytes as binary",
			input: string(
				[]byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00},
			), // WASM magic as string
			expectedType: (*FromString)(
				nil,
			), // Raw binary should be string, not auto-detected
		},
		{
			name:         "content with multiple spaces",
			input:        "const  x  =  'value';",
			expectedType: (*FromString)(nil),
		},
		{
			name:         "windows path with spaces and unsupported extension",
			input:        "C:\\Program Files\\App with spaces\\script.invalid",
			expectedType: (*FromString)(nil),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := InferLoader(tc.input)
			require.NoError(t, err)
			assert.IsType(t, tc.expectedType, result, "Input: %q", tc.input)
		})
	}
}
