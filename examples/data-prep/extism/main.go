package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/extism"
	"github.com/robbyt/go-polyscript/options"
)

// FindWasmFile searches for the Extism WASM file in various likely locations
func FindWasmFile(logger *slog.Logger) (string, error) {
	paths := []string{
		"main.wasm",                                            // Current directory
		"extism/main.wasm",                                     // extism subdirectory
		"data-prep/extism/main.wasm",                           // data-prep/extism subdirectory
		"examples/extism/main.wasm",                            // examples/extism
		"examples/data-prep/extism/main.wasm",                  // examples/data-prep/extism
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

// simulateAPIRequest creates fake data to simulate an incoming API request
func simulateAPIRequest() map[string]any {
	return map[string]any{
		"input":     "World",
		"timestamp": time.Now().Format(time.RFC3339),
		"requestId": fmt.Sprintf("req-%d", time.Now().UnixNano()),
	}
}

// simulateConfigData creates configuration data that might be loaded from a database
func simulateConfigData() map[string]any {
	return map[string]any{
		"mode": "production",
		"features": map[string]bool{
			"advanced_greeting": true,
			"timestamps":        true,
		},
	}
}

// demonstrateAsyncPreparation shows how data preparation can be done asynchronously
// before evaluation, which is useful in high-throughput systems
func demonstrateAsyncPreparation(evaluator engine.EvaluatorWithPrep, logger *slog.Logger) (map[string]any, error) {
	// Base context
	ctx := context.Background()

	// Channel to send prepared contexts between goroutines
	ctxChan := make(chan struct {
		ctx context.Context
		err error
	}, 1)

	// Step 1: Asynchronously prepare the context in a separate goroutine
	// -----------------------------------------------------------------
	logger.Info("Starting async context preparation")

	go func() {
		// Create a local context with timeout for the preparation process
		prepCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		logger.Info("Async routine: Gathering data")
		// Simulate gathering data from different sources
		apiData := simulateAPIRequest()
		configData := simulateConfigData()

		// First preparation step with API data
		enrichedCtx, err := evaluator.PrepareContext(prepCtx, apiData)
		if err != nil {
			logger.Error("Async routine: Failed to prepare context with API data", "error", err)
			ctxChan <- struct {
				ctx context.Context
				err error
			}{ctx: nil, err: err}
			return
		}

		// Second preparation step with config data
		finalCtx, err := evaluator.PrepareContext(enrichedCtx, configData)
		if err != nil {
			logger.Error("Async routine: Failed to prepare context with config data", "error", err)
			ctxChan <- struct {
				ctx context.Context
				err error
			}{ctx: nil, err: err}
			return
		}

		logger.Info("Async routine: Context preparation complete")
		// Send the fully prepared context through the channel
		ctxChan <- struct {
			ctx context.Context
			err error
		}{ctx: finalCtx, err: nil}
	}()

	// Step 2: Wait for the preparation to complete and evaluate
	// --------------------------------------------------------
	logger.Info("Main routine: Waiting for context preparation")

	// Receive the prepared context or error
	result := <-ctxChan
	if result.err != nil {
		return nil, fmt.Errorf("context preparation failed: %w", result.err)
	}

	// Set a timeout for the evaluation
	evalCtx, cancel := context.WithTimeout(result.ctx, 5*time.Second)
	defer cancel()

	// Step 3: Evaluate the script with the prepared context
	// ----------------------------------------------------
	logger.Info("Main routine: Evaluating script with prepared context")

	// Evaluate the script
	response, err := evaluator.Eval(evalCtx)
	if err != nil {
		logger.Error("Main routine: Failed to evaluate script", "error", err)
		return nil, fmt.Errorf("script evaluation failed: %w", err)
	}

	// Process the result
	val := response.Interface()
	if val == nil {
		logger.Warn("Main routine: Result is nil")
		return map[string]any{}, nil
	}

	// Return the result
	resultMap, ok := val.(map[string]any)
	if !ok {
		logger.Error("Main routine: Unexpected response type", "type", fmt.Sprintf("%T", val))
		return nil, fmt.Errorf("unexpected response type: %T", val)
	}

	return resultMap, nil
}

func main() {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler.WithGroup("extism-data-prep-example"))

	// Find the WASM file
	wasmFilePath, err := FindWasmFile(logger)
	if err != nil {
		logger.Error("Failed to find WASM file", "error", err)
		os.Exit(1)
	}

	// Create data provider - we'll use a context provider for this example
	dataProvider := data.NewContextProvider("eval_data")

	// Create evaluator using the functional options pattern
	evaluator, err := polyscript.FromExtismFile(
		wasmFilePath,
		options.WithLogger(handler),
		options.WithDataProvider(dataProvider),
		extism.WithEntryPoint("greet"),
	)
	if err != nil {
		logger.Error("Failed to create evaluator", "error", err)
		os.Exit(1)
	}

	// Verify the evaluator is properly initialized
	if evaluator == nil {
		logger.Error("Evaluator is nil")
		os.Exit(1)
	}

	// Run the asynchronous preparation example
	result, err := demonstrateAsyncPreparation(evaluator, logger)
	if err != nil {
		logger.Error("Failed to run async preparation example", "error", err)
		os.Exit(1)
	}

	// Print the result
	logger.Info("Final result", "data", result)
}
