package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript"
)

//go:embed testdata/script.star
var starlarkScript string

// runStarlarkExample executes a Starlark script once and returns the result
func runStarlarkExample(logger *slog.Logger) (map[string]any, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Create input data
	input := map[string]any{
		"name": "World",
	}

	// Create evaluator using the new simplified interface
	evaluator, err := polyscript.FromStarlarkStringWithData(
		starlarkScript,
		input,
		logger.Handler(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluator: %w", err)
	}

	// Execute the script
	ctx := context.Background()
	result, err := evaluator.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate script: %w", err)
	}

	// Handle potential nil result from Interface()
	val := result.Interface()
	if val == nil {
		logger.Warn("Result is nil")
		return map[string]any{}, nil
	}

	// Process the result
	data, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("result is not a map: %T", val)
	}
	return data, nil
}

func run() error {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, nil)
	logger := slog.New(handler.WithGroup("starlark-simple-example"))

	// Run the example
	result, err := runStarlarkExample(logger)
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
