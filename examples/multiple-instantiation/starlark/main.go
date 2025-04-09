package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/engine/options"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/starlark/compiler"
)

// StarlarkEvaluator is a type alias to make testing cleaner
type StarlarkEvaluator = engine.EvaluatorWithPrep

//go:embed testdata/script.star
var starlarkScript string

// createEvaluator initializes a Starlark evaluator with context provider for runtime data
func createEvaluator(handler slog.Handler) (StarlarkEvaluator, error) {
	if handler == nil {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}
	logger := slog.New(handler)

	// Define globals that will be available to the script
	globals := []string{constants.Ctx}

	// Create a context provider for runtime data
	ctxProvider := data.NewContextProvider(constants.EvalData)

	// Create evaluator using the functional options pattern
	evaluator, err := polyscript.FromStarlarkString(
		starlarkScript,
		options.WithDefaults(),
		options.WithLogHandler(handler),
		options.WithDataProvider(ctxProvider),
		compiler.WithGlobals(globals),
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
			logger.Error("Script evaluation failed", "error", err, "name", name)
			continue
		}

		// Process result
		val := result.Interface()
		if val == nil {
			logger.Warn("Result is nil for name", "name", name)
			continue
		}

		data, ok := val.(map[string]any)
		if !ok {
			logger.Error("Result is not a map", "type", fmt.Sprintf("%T", val), "name", name)
			continue
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
