package helpers

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// FindWasmFile searches for an Extism WASM file in various likely locations
// It searches in the current directory, project examples, and engines directories
//
// Parameters:
//   - logger: Optional logger for verbose output
//
// Returns:
//   - Absolute path to the found WASM file
//   - Error if no file is found with detailed information about searched paths
func FindWasmFile(logger *slog.Logger) (string, error) {
	paths := []string{
		"main.wasm",                   // Current directory
		"examples/testdata/main.wasm", // Project's main example WASM
		"../../../engines/extism/testdata/examples/main.wasm", // From engines testdata
		"engines/extism/testdata/examples/main.wasm",          // From project root to testdata
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
	errMsg := `WASM file not found in any of the expected locations.

To fix this issue:
1. Run 'make build' in the engines/extism/testdata directory to generate the WASM file
2. OR copy a pre-built WASM file to one of these locations:
`

	for _, path := range checkedPaths {
		errMsg += "   - " + path + "\n"
	}

	return "", fmt.Errorf("%s", errMsg)
}
