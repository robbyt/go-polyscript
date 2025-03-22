package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/extism"
	"github.com/robbyt/go-polyscript/options"
)

// FindWasmFile searches for the Extism WASM file in various likely locations
func FindWasmFile(logger *slog.Logger) (string, error) {
	paths := []string{
		"main.wasm",                          // Current directory
		"multiple/main.wasm",                 // multiple subdirectory
		"extism/multiple/main.wasm",          // extism/multiple subdirectory
		"examples/extism/multiple/main.wasm", // examples/extism/multiple subdirectory
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

// RunExtismMultipleTimes demonstrates the "compile once, run many times" pattern with Extism
func RunExtismMultipleTimes(handler slog.Handler) ([]map[string]any, error) {
	if handler == nil {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	logger := slog.New(handler.WithGroup("extism-multiple-example"))

	// Find the WASM file
	wasmFilePath, err := FindWasmFile(logger)
	if err != nil {
		logger.Error("Failed to find WASM file", "error", err)
		return nil, err
	}

	// Create the context provider for dynamic data
	dataProvider := data.NewContextProvider(constants.EvalData)

	// Create the evaluator
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

	// Inputs to process - we'll run the script once for each input
	inputs := []string{"World", "Extism", "Go-PolyScript"}
	results := make([]map[string]any, 0, len(inputs))

	// Execute the script multiple times with different inputs
	for i, input := range inputs {
		// Create input data for this execution
		contextData := map[string]any{
			"input": input,
		}

		// Create a context with our input data
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		// Important: Use the EvalData constant for context value key to match the ContextProvider
		ctx = context.WithValue(ctx, constants.EvalData, contextData)

		// Execute the script
		response, err := evaluator.Eval(ctx)
		cancel() // Cancel the context after execution

		if err != nil {
			logger.Error("Script evaluation failed",
				"error", err,
				"evaluator", fmt.Sprintf("%T", evaluator),
				"execution", i+1)
			return nil, err
		}

		// Get the result as a map
		val := response.Interface()
		if val == nil {
			logger.Warn("Result is nil", "execution", i+1)
			continue
		}

		data, ok := val.(map[string]any)
		if !ok {
			logger.Error("Result is not a map",
				"type", fmt.Sprintf("%T", val),
				"execution", i+1)
			return nil, fmt.Errorf("result is not a map: %T", val)
		}

		// Add the result to the list
		results = append(results, data)
	}

	return results, nil
}

func main() {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler.WithGroup("main"))

	// Run the example
	results, err := RunExtismMultipleTimes(handler)
	if err != nil {
		logger.Error("Failed to run Extism example", "error", err)
		os.Exit(1)
	}

	// Print the results
	for i, result := range results {
		logger.Info("Result", "index", i+1, "output", result)
	}
}
