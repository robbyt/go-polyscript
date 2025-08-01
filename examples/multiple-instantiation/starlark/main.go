package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/constants"
)

// StarlarkEvaluator is a type alias to make testing cleaner
type StarlarkEvaluator = platform.Evaluator

//go:embed testdata/script.star
var starlarkScript string

// createEvaluator initializes a Starlark evaluator with context provider for runtime data
func createEvaluator(logger *slog.Logger) (StarlarkEvaluator, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Create evaluator using the new simplified interface
	// This provides a dynamic context provider automatically
	evaluator, err := polyscript.FromStarlarkString(
		starlarkScript,
		logger.Handler(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluator: %w", err)
	}

	return evaluator, nil
}

// runMultipleTimes demonstrates the "compile once, run many times" pattern
func runMultipleTimes(logger *slog.Logger) ([]map[string]any, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Create the evaluator once
	evaluator, err := createEvaluator(logger)
	if err != nil {
		return nil, err
	}

	// Names to greet - we'll run the script once for each name
	names := []string{"World", "Alice", "Bob", "Charlie"}
	results := make([]map[string]any, 0, len(names))

	// Execute the script multiple times with different input data
	for _, name := range names {
		// Create context data for this execution
		contextData := map[string]any{
			"name": name,
		}

		// Create a context with the specific data for this run
		ctx := context.Background()
		ctx = context.WithValue(ctx, constants.EvalData, contextData)

		logger.Debug("Running script with name", "name", name)

		// Execute the script with the context
		result, err := evaluator.Eval(ctx)
		if err != nil {
			return nil, fmt.Errorf("script evaluation failed for name %q: %w", name, err)
		}

		// Process result
		val := result.Interface()
		if val == nil {
			logger.Warn("Result is nil for name", "name", name)
			continue
		}

		data, ok := val.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("result is not a map for name %q: %T", name, val)
		}

		results = append(results, data)
		logger.Info("Processed result", "name", name, "greeting", data["greeting"])
	}

	return results, nil
}

func run() error {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, nil)
	logger := slog.New(handler.WithGroup("starlark-multiple-example"))

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
