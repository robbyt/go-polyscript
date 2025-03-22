package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/extism"
	"github.com/robbyt/go-polyscript/options"
)

// FindWasmFile searches for the Extism WASM file in various likely locations
func FindWasmFile(logger *slog.Logger) (string, error) {
	paths := []string{
		"main.wasm",                        // Current directory
		"simple/main.wasm",                 // simple subdirectory
		"extism/simple/main.wasm",          // extism/simple subdirectory
		"examples/extism/simple/main.wasm", // examples/extism/simple subdirectory
		"../../../machines/extism/testdata/examples/main.wasm", // From machines testdata
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

// RunExtismExample executes an Extism WASM module and returns the result
func RunExtismExample(handler slog.Handler) (map[string]any, error) {
	if handler == nil {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	logger := slog.New(handler.WithGroup("extism-simple-example"))

	// Find the WASM file
	wasmFilePath, err := FindWasmFile(logger)
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
		options.WithLogger(handler),
		options.WithDataProvider(dataProvider),
		extism.WithEntryPoint("greet"),
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

	// Return the result
	result, ok := response.Interface().(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", response.Interface())
	}
	return result, nil
}

func main() {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler.WithGroup("main"))

	// Run the example
	result, err := RunExtismExample(handler)
	if err != nil {
		logger.Error("Failed to run Extism example", "error", err)
		os.Exit(1)
	}

	// Print the result
	logger.Info("Result", "data", result)
}
