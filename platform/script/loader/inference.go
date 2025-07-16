package loader

import (
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
)

// InferLoader analyzes the input and returns an appropriate loader based on type inference.
// It supports the following input types:
//   - string: Infers from URI scheme (http/https -> HTTP, file -> Disk) or content (inline string with base64 decoding)
//   - []byte: Returns FromBytes loader
//   - io.Reader: Returns FromIoReader loader
//   - Loader: Returns as-is
//
// Returns an error if the input type is unsupported or if loader creation fails.
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
// It checks for URI schemes and falls back to treating the string as inline content.
func inferFromString(input string) (Loader, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty string input")
	}

	// Try to parse as URL to detect scheme
	if parsed, err := url.Parse(input); err == nil && parsed.Scheme != "" {
		switch parsed.Scheme {
		case "http", "https":
			return NewFromHTTP(input)
		case "file":
			path := parsed.Path
			// Convert relative paths to absolute paths
			if !filepath.IsAbs(path) {
				absPath, err := filepath.Abs(path)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve relative path %q: %w", path, err)
				}
				path = absPath
			}
			return NewFromDisk(path)
		default:
			// Unknown scheme, treat as inline content
			return NewFromStringBase64(input)
		}
	}

	// Check if it looks like a file path
	if filepath.IsAbs(input) || strings.Contains(input, "/") || strings.Contains(input, "\\") {
		// Convert relative paths to absolute paths
		if !filepath.IsAbs(input) {
			absPath, err := filepath.Abs(input)
			if err != nil {
				// If we can't resolve the path, treat as inline content
				return NewFromStringBase64(input)
			}
			input = absPath
		}
		return NewFromDisk(input)
	}

	// Default to treating as inline string content
	return NewFromStringBase64(input)
}
