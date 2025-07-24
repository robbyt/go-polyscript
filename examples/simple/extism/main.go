package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
)

// runExtismExample executes an Extism WASM module and returns the result
func runExtismExample(logger *slog.Logger) (map[string]any, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Create input data
	inputData := map[string]any{
		"input": "World",
	}

	// Create evaluator using embedded WASM
	evaluator, err := polyscript.FromExtismBytesWithData(
		wasmdata.TestModule,
		inputData,
		logger.Handler(),
		wasmdata.EntrypointGreet,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluator: %w", err)
	}

	// Set a timeout for script execution
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Evaluate the script
	response, err := evaluator.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate script: %w", err)
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
	result, err := runExtismExample(logger)
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
