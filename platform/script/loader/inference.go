package loader

import (
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
)

// InferLoader determines the appropriate loader for the given input.
//
// For string inputs, it applies the following checks in order:
//  1. URI scheme detection (http/https -> HTTP, file -> Disk)
//  2. File path format detection (absolute paths, known extensions)
//  3. Base64 validation (attempts decode)
//  4. String fallback
//
// Other input types:
//   - []byte: FromBytes loader
//   - io.Reader: FromIoReader loader
//   - Loader: returned unchanged
//
// Returns an error for unsupported input types or loader creation failures.
func InferLoader(input any) (Loader, error) {
	switch v := input.(type) {
	case string:
		return inferFromString(v)
	case []byte:
		return NewFromBytes(v)
	case io.Reader:
		return NewFromIoReader(v, "inferred")
	case Loader:
		return v, nil
	default:
		return nil, fmt.Errorf("unsupported input type: %T", input)
	}
}

// inferFromString analyzes a string input and returns the appropriate loader.
// It follows this order: scheme check -> file path check -> base64 check -> string fallback.
func inferFromString(input string) (Loader, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty string input")
	}

	// 1. Check for URI scheme
	if parsed, err := url.Parse(input); err == nil && parsed.Scheme != "" {
		switch parsed.Scheme {
		case "http", "https":
			return NewFromHTTP(input)
		case "file":
			path := parsed.Path
			if !filepath.IsAbs(path) {
				absPath, err := filepath.Abs(path)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve relative path %q: %w", path, err)
				}
				path = absPath
			}
			return NewFromDisk(path)
		}
	}

	// 2. Check if it's a valid file path
	if isValidFilePath(input) {
		path := input
		// Convert relative paths to absolute paths
		if !filepath.IsAbs(input) {
			absPath, err := filepath.Abs(input)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve relative path %q: %w", input, err)
			}
			path = absPath
		}
		return NewFromDisk(path)
	}

	// 3. Use the base64 check to determine if the input is valid base64
	// (it returns FromBytes if valid, otherwise falls back to FromString)
	return NewFromStringBase64(input)
}

// isValidFilePath checks if a string looks like a valid file path format.
func isValidFilePath(s string) bool {
	// Don't consider strings with newlines/carriage returns as file paths
	if strings.ContainsAny(s, "\n\r") {
		return false
	}

	// Don't consider strings that look like code as file paths
	if strings.Contains(s, " -c ") || // shell commands
		strings.Contains(s, "';") || // code with semicolons after quotes
		strings.Contains(s, "console.") || // JavaScript console calls
		strings.Contains(s, "var ") || // variable declarations
		strings.Contains(s, "function ") { // function declarations
		return false
	}

	// Check for specific script file extensions (case insensitive)
	validExtensions := []string{".wasm", ".risor", ".star", ".starlark"}
	lower := strings.ToLower(s)
	for _, ext := range validExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}

	// Don't treat absolute paths as file paths unless they have supported extensions
	return false
}
