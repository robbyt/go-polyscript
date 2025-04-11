package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/constants"
)

// RisorEvaluator is a type alias to make testing cleaner
type RisorEvaluator = engine.EvaluatorWithPrep

//go:embed testdata/script.risor
var risorScript string

// createEvaluator initializes a Risor evaluator with context provider for runtime data
func createEvaluator(handler slog.Handler) (RisorEvaluator, error) {
	if handler == nil {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}
	logger := slog.New(handler)

	// Create evaluator using the new simplified interface
	// This provides a dynamic context provider automatically
	evaluator, err := polyscript.FromRisorString(
		risorScript,
		handler,
	)
	if err != nil {
		logger.Error("Failed to create evaluator", "error", err)
		return nil, err
	}

	return evaluator, nil
}

// runMultipleTimes demonstrates the "compile once, run many times" pattern
func runMultipleTimes(handler slog.Handler) ([]map[string]any, error) {
	if handler == nil {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}
	logger := slog.New(handler)

	// Create the evaluator once
	evaluator, err := createEvaluator(handler)
	if err != nil {
		return nil, err
	}

	// Names to greet - we'll run the script once for each name
	names := []string{"World", "Risor", "Go"}
	results := make([]map[string]any, 0, len(names))

	// Execute the script multiple times with different inputs
	for i, name := range names {
		// Create context data for this execution
		contextData := map[string]any{
			"name": name,
		}

		// Create a context with the specific data for this run
		ctx := context.Background()
		ctx = context.WithValue(ctx, constants.EvalData, contextData)

		logger.Debug("Running script with name", "name", name, "execution", i+1)

		// Execute the script with the context
		result, err := evaluator.Eval(ctx)
		if err != nil {
			logger.Error("Script evaluation failed", "error", err, "execution", i+1)
			return nil, err
		}

		// Process result
		val := result.Interface()
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
		logger.Info("Processed result", "name", name, "greeting", data["greeting"])
	}

	return results, nil
}

func run() error {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, nil)
	logger := slog.New(handler.WithGroup("risor-multiple-example"))

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
