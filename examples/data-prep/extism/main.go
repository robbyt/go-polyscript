// Package main provides an example of using Extism with go-polyscript
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
)

// ExtismEvaluator is a type alias to make testing cleaner
type ExtismEvaluator = platform.Evaluator

// prepareRuntimeData adds dynamic runtime data to the context.
// Returns the enriched context or an error.
func prepareRuntimeData(
	ctx context.Context,
	logger *slog.Logger,
	evaluator ExtismEvaluator,
) (context.Context, error) {
	logger.Info("Preparing runtime data")

	// Create user data
	userData := map[string]any{
		"id":          "user-123",
		"role":        "admin",
		"permissions": "read,write,execute",
	}

	// For Extism, we need to structure our data so that the "input" field
	// will be directly accessible to the WASM module
	requestMeta := map[string]any{
		"input":     "World", // This is what the Extism WASM module needs
		"user_data": userData,
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	}

	// Add the request metadata to the context using the data.Provider
	enrichedCtx, err := evaluator.AddDataToContext(ctx, requestMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare context: %w", err)
	}

	logger.Debug("Runtime data prepared successfully")
	return enrichedCtx, nil
}

// evalAndExtractResult evaluates the script with the prepared context.
// Returns the result as a map[string]any or an error.
func evalAndExtractResult(
	ctx context.Context,
	logger *slog.Logger,
	evaluator ExtismEvaluator,
) (map[string]any, error) {
	logger.Info("Evaluating script")

	// Set a timeout for the evaluation
	evalCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Evaluate the script
	response, err := evaluator.Eval(evalCtx)
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
	resultMap, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", val)
	}

	logger.Info("Script evaluated successfully")
	return resultMap, nil
}

func run() error {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler.WithGroup("extism-data-prep-example"))

	// Static data loaded into a data provider
	staticData := map[string]any{
		"app_version": "1.0.0",
		"environment": "development",
		"config": map[string]any{
			"timeout":     30,
			"max_retries": 3,
			"feature_flags": map[string]any{
				"advanced_features": true,
				"beta_features":     false,
			},
		},
		// Put the input field directly at the top level for Extism
		"input": "Static User",
	}

	// Create evaluator with embedded WASM and static data
	evaluator, err := polyscript.FromExtismBytesWithData(
		wasmdata.TestModule,
		staticData,
		logger.Handler(),
		wasmdata.EntrypointGreet,
	)
	if err != nil {
		return fmt.Errorf("failed to create evaluator: %w", err)
	}

	// Prepare runtime data
	ctx, err := prepareRuntimeData(context.Background(), logger, evaluator)
	if err != nil {
		return fmt.Errorf("failed to prepare context: %w", err)
	}

	// Run the example
	result, err := evalAndExtractResult(ctx, logger, evaluator)
	if err != nil {
		return fmt.Errorf("failed to run example: %w", err)
	}

	// Print the result
	logger.Info("Final result", "data", result)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	fmt.Println("Success")
}
