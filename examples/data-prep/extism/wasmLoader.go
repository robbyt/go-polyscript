package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// FindWasmFile searches for the Extism WASM file in various likely locations
func FindWasmFile(logger *slog.Logger) (string, error) {
	paths := []string{
		"main.wasm",                                            // Current directory
		"examples/main.wasm",                                   // Project's main example WASM
		"extism/main.wasm",                                     // extism subdirectory
		"data-prep/extism/main.wasm",                           // data-prep/extism subdirectory
		"examples/extism/main.wasm",                            // examples/extism
		"examples/data-prep/extism/main.wasm",                  // examples/data-prep/extism
		"../../../machines/extism/testdata/examples/main.wasm", // From machines testdata
		"machines/extism/testdata/examples/main.wasm",          // From project root to testdata
	}

	// Log the searched paths if logger is available
	if logger != nil {
		logger.Info("Searching for WASM file in multiple locations")
	}

	// Track checked paths for better error reporting
	checkedPaths := []string{}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			absPath, err := filepath.Abs(path)
			if err == nil {
				if logger != nil {
					logger.Info("Found WASM file", "path", absPath)
				}
				return absPath, nil
			}
		}
		// Store the absolute path for error reporting
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path // Fallback to relative path if absolute path fails
		}
		checkedPaths = append(checkedPaths, absPath)
	}

	// If no WASM file found, provide detailed error with checked paths
	errMsg := "WASM file not found in any of the expected locations.\n\nTo fix this issue:\n" +
		"1. Run 'make build' in the machines/extism/testdata directory to generate the WASM file\n" +
		"2. OR copy a pre-built WASM file to one of these locations:\n"

	for _, path := range checkedPaths {
		errMsg += "   - " + path + "\n"
	}

	return "", fmt.Errorf("%s", errMsg)
}
