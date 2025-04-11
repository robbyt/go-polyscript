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
func runStarlarkExample(handler slog.Handler) (map[string]any, error) {
	if handler == nil {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}
	logger := slog.New(handler)

	// Create input data
	input := map[string]any{
		"name": "World",
	}

	// Create evaluator using the new simplified interface
	evaluator, err := polyscript.FromStarlarkStringWithData(
		starlarkScript,
		input,
		handler,
	)
	if err != nil {
		logger.Error("Failed to create evaluator", "error", err)
		return nil, err
	}

	// Execute the script
	ctx := context.Background()
	result, err := evaluator.Eval(ctx)
	if err != nil {
		logger.Error("Failed to evaluate script", "error", err)
		return nil, err
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
		logger.Error("Result is not a map", "type", fmt.Sprintf("%T", val))
		return nil, fmt.Errorf("result is not a map: %T", val)
	}
	return data, nil
}

func run() error {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, nil)
	logger := slog.New(handler.WithGroup("starlark-simple-example"))

	// Run the example
	result, err := runStarlarkExample(handler)
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
