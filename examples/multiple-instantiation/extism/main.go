package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/constants"
)

// ExtismEvaluator is a type alias to make testing cleaner
type ExtismEvaluator = platform.Evaluator

// createEvaluator initializes an Extism evaluator with context provider for runtime data
func createEvaluator(logger *slog.Logger) (ExtismEvaluator, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Create the evaluator using embedded WASM
	// Uses the simpler interface with dynamic data only via context
	evaluator, err := polyscript.FromExtismBytes(
		wasmdata.TestModule,
		logger.Handler(),
		wasmdata.EntrypointGreet,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluator: %w", err)
	}

	return evaluator, nil
}

// runMultipleTimes demonstrates the "compile once, run many times" pattern with Extism
func runMultipleTimes(logger *slog.Logger) ([]map[string]any, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Create the evaluator once
	evaluator, err := createEvaluator(logger)
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
			return nil, fmt.Errorf(
				"evaluation failed for input %q (execution %d): %w",
				input,
				i+1,
				err,
			)
		}

		// Process result
		val := response.Interface()
		if val == nil {
			logger.Warn("Result is nil", "execution", i+1)
			continue
		}

		data, ok := val.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("result is not a map for execution %d: %T", i+1, val)
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
	results, err := runMultipleTimes(logger)
	if err != nil {
		return fmt.Errorf("failed to run example: %w", err)
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
