package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/abstract/constants"
	"github.com/robbyt/go-polyscript/abstract/evaluation"
	"github.com/robbyt/go-polyscript/internal/helpers"
)

// ExtismEvaluator is a type alias to make testing cleaner
type ExtismEvaluator = evaluation.Evaluator

// createEvaluator initializes an Extism evaluator with context provider for runtime data
func createEvaluator(handler slog.Handler) (ExtismEvaluator, error) {
	if handler == nil {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	logger := slog.New(handler.WithGroup("extism-evaluator"))

	// Find the WASM file
	wasmFilePath, err := helpers.FindWasmFile(logger)
	if err != nil {
		logger.Error("Failed to find WASM file", "error", err)
		return nil, err
	}

	// Create the evaluator
	// Uses the simpler interface with dynamic data only via context
	evaluator, err := polyscript.FromExtismFile(
		wasmFilePath,
		handler,
		"greet", // entry point
	)
	if err != nil {
		logger.Error("Failed to create evaluator", "error", err)
		return nil, err
	}

	return evaluator, nil
}

// runMultipleTimes demonstrates the "compile once, run many times" pattern with Extism
func runMultipleTimes(handler slog.Handler) ([]map[string]any, error) {
	if handler == nil {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	logger := slog.New(handler.WithGroup("extism-multiple"))

	// Create the evaluator once
	evaluator, err := createEvaluator(handler)
	if err != nil {
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
		ctx = context.WithValue(ctx, constants.EvalData, contextData)

		logger.Debug("Running WebAssembly with input", "input", input, "execution", i+1)

		// Execute the WebAssembly module
		response, err := evaluator.Eval(ctx)
		cancel() // Cancel the context after execution

		if err != nil {
			logger.Error("Evaluation failed", "error", err, "execution", i+1)
			return nil, err
		}

		// Process result
		val := response.Interface()
		if val == nil {
			logger.Warn("Result is nil", "execution", i+1)
			continue
		}

		data, ok := val.(map[string]any)
		if !ok {
			logger.Error("Result is not a map", "type", fmt.Sprintf("%T", val), "execution", i+1)
			return nil, fmt.Errorf("result is not a map: %T", val)
		}

		results = append(results, data)
		logger.Info("Processed result", "input", input, "output", data)
	}

	return results, nil
}

func run() error {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler.WithGroup("extism-example"))

	// Run the example
	results, err := runMultipleTimes(handler)
	if err != nil {
		logger.Error("Failed to run example", "error", err)
		return err
	}

	// Print the results
	for i, result := range results {
		logger.Info("Result", "index", i+1, "data", result)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
