package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engine/options"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/extism/compiler"
)

// findWasmFile searches for the Extism WASM file in various likely locations
func findWasmFile(logger *slog.Logger) (string, error) {
	paths := []string{
		"main.wasm",                   // Current directory
		"examples/testdata/main.wasm", // Project's main example WASM
		"../../../machines/extism/testdata/examples/main.wasm", // From machines testdata
		"machines/extism/testdata/examples/main.wasm",          // From project root to testdata
	}

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
	}

	return "", fmt.Errorf("WASM file not found in any of the expected locations")
}

// runExtismExample executes an Extism WASM module and returns the result
func runExtismExample(handler slog.Handler) (map[string]any, error) {
	if handler == nil {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	logger := slog.New(handler)

	// Find the WASM file
	wasmFilePath, err := findWasmFile(logger)
	if err != nil {
		logger.Error("Failed to find WASM file", "error", err)
		return nil, err
	}

	// Create input data
	inputData := map[string]any{
		"input": "World",
	}
	dataProvider := data.NewStaticProvider(inputData)

	// Create evaluator using the functional options pattern
	evaluator, err := polyscript.FromExtismFile(
		wasmFilePath,
		options.WithDefaults(),
		options.WithLogHandler(handler),
		options.WithDataProvider(dataProvider),
		compiler.WithEntryPoint("greet"),
	)
	if err != nil {
		logger.Error("Failed to create evaluator", "error", err)
		return nil, err
	}

	// Set a timeout for script execution
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Evaluate the script
	response, err := evaluator.Eval(ctx)
	if err != nil {
		logger.Error("Failed to evaluate script", "error", err)
		return nil, err
	}

	// Process the result
	val := response.Interface()
	if val == nil {
		logger.Warn("Result is nil")
		return map[string]any{}, nil
	}

	// Return the result
	result, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", val)
	}
	return result, nil
}

func run() error {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler.WithGroup("extism-simple-example"))

	// Run the example
	result, err := runExtismExample(handler)
	if err != nil {
		return fmt.Errorf("failed to run example: %w", err)
	}

	// Print the result
	logger.Info("Result", "data", result)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
