package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/extism"
	"github.com/robbyt/go-polyscript/options"
)

// ExtismEvaluator is a type alias to make testing cleaner
type ExtismEvaluator = engine.EvaluatorWithPrep

// CreateExtismEvaluator creates a new Extism evaluator with the given WASM file and logger
func CreateExtismEvaluator(wasmFilePath string, handler slog.Handler) (*ExtismEvaluator, error) {
	// Create a StaticProvider with compile-time data
	staticData := map[string]any{
		// Put the input field directly at the top level for Extism
		"input": "API User",
	}
	staticProvider := data.NewStaticProvider(staticData)

	// Also create a ContextProvider for any runtime data
	// Using the eval_data key where data will be stored in the context
	ctxProvider := data.NewContextProvider("eval_data")

	// Create a CompositeProvider that combines both static and runtime data
	// The ordering is important: runtime data (ctxProvider) will override static data
	compositeProvider := data.NewCompositeProvider(staticProvider, ctxProvider)

	// Create evaluator using the functional options pattern
	evaluator, err := polyscript.FromExtismFile(
		wasmFilePath,
		options.WithLogger(handler),
		options.WithDataProvider(compositeProvider), // Use the composite provider
		extism.WithEntryPoint("greet"),
	)
	if err != nil {
		return nil, err
	}

	return &evaluator, nil
}

// PrepareRequestData prepares the context with request data
func PrepareRequestData(
	ctx context.Context,
	evaluator ExtismEvaluator,
	logger *slog.Logger,
) (context.Context, error) {
	// Simulate gathering data from an API request
	logger.Info("Preparing request data")

	// For Extism, we need to structure our data so that the "input" field
	// will be directly accessible to the WASM module. The ContextProvider wraps
	// map[string]any data in script_data, so we need to add our data there.
	scriptData := map[string]any{
		"input": "API User", // This is what the Extism WASM module needs
	}

	// Enrich context with request data
	enrichedCtx, err := evaluator.PrepareContext(ctx, scriptData)
	if err != nil {
		logger.Error("Failed to prepare context with request data", "error", err)
		return nil, err
	}

	logger.Info("Request data prepared successfully")
	return enrichedCtx, nil
}

// PrepareConfigData prepares the context with configuration data
func PrepareConfigData(
	ctx context.Context,
	evaluator ExtismEvaluator,
	logger *slog.Logger,
) (context.Context, error) {
	// For Extism, we'll keep using the simple structure since the WASM module
	// only expects {"input": "..."} format
	// This function is kept for consistency with other examples and to demonstrate
	// the multi-step preparation pattern
	logger.Info("Preparing configuration data")

	// We don't need to add any additional data for this example
	enrichedCtx := ctx

	logger.Info("Configuration data prepared successfully")
	return enrichedCtx, nil
}

// EvaluateScript evaluates the script with the prepared context
func EvaluateScript(
	ctx context.Context,
	evaluator ExtismEvaluator,
	logger *slog.Logger,
) (map[string]any, error) {
	logger.Info("Evaluating script")

	// Set a timeout for the evaluation
	evalCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Evaluate the script
	response, err := evaluator.Eval(evalCtx)
	if err != nil {
		logger.Error("Failed to evaluate script", "error", err)
		return nil, err
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
		logger.Error("Unexpected response type", "type", fmt.Sprintf("%T", val))
		return nil, fmt.Errorf("unexpected response type: %T", val)
	}

	logger.Info("Script evaluated successfully")
	return resultMap, nil
}

// DemonstrateDataPrepAndEval shows the complete data preparation and evaluation workflow
func DemonstrateDataPrepAndEval(
	evaluator ExtismEvaluator,
	logger *slog.Logger,
) (map[string]any, error) {
	// Create a base context
	ctx := context.Background()

	// Step 1: Prepare context with request data
	ctx, err := PrepareRequestData(ctx, evaluator, logger)
	if err != nil {
		return nil, fmt.Errorf("request data preparation failed: %w", err)
	}

	// Step 2: Prepare context with config data
	ctx, err = PrepareConfigData(ctx, evaluator, logger)
	if err != nil {
		return nil, fmt.Errorf("config data preparation failed: %w", err)
	}

	// Step 3: Evaluate the script with the prepared context
	result, err := EvaluateScript(ctx, evaluator, logger)
	if err != nil {
		return nil, fmt.Errorf("script evaluation failed: %w", err)
	}

	return result, nil
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

	// Create evaluator
	evaluator, err := CreateExtismEvaluator(wasmFilePath, handler)
	if err != nil {
		logger.Error("Failed to create evaluator", "error", err)
		os.Exit(1)
	}

	// Run the data preparation and evaluation example
	result, err := DemonstrateDataPrepAndEval(*evaluator, logger)
	if err != nil {
		logger.Error("Failed to run data prep and eval example", "error", err)
		os.Exit(1)
	}

	// Print the result
	logger.Info("Final result", "data", result)
}
