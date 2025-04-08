// Package main provides an example of using Extism with go-polyscript
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
	"github.com/robbyt/go-polyscript/engine/options"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/extism/compiler"
)

// ExtismEvaluator is a type alias to make testing cleaner
type ExtismEvaluator = engine.EvaluatorWithPrep

// createExtismEvaluator creates a new Extism evaluator with the given WASM file and logger.
// Sets up a CompositeProvider that combines static and dynamic data providers.
func createExtismEvaluator(
	logger *slog.Logger,
	wasmFilePath string,
	staticData map[string]any,
) (ExtismEvaluator, error) {
	// The static provider enables access to the static data map
	staticProvider := data.NewStaticProvider(staticData)

	// This context provider enables each request to add different dynamic data
	dynamicProvider := data.NewContextProvider(constants.EvalData)

	// Composite provider handles static data first, then dynamic data
	compositeProvider := data.NewCompositeProvider(staticProvider, dynamicProvider)

	// Create evaluator using the functional options pattern
	return polyscript.FromExtismFile(
		wasmFilePath,
		options.WithLogHandler(logger.Handler()),
		options.WithDataProvider(compositeProvider),
		compiler.WithEntryPoint("greet"),
	)
}

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
	enrichedCtx, err := evaluator.PrepareContext(ctx, requestMeta)
	if err != nil {
		logger.Error("Failed to prepare context with request data", "error", err)
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

// findWasmFile searches for the Extism WASM file in various likely locations
func findWasmFile(logger *slog.Logger) (string, error) {
	paths := []string{
		"main.wasm",                   // Current directory
		"examples/testdata/main.wasm", // Project's main example WASM
		"../../../machines/extism/testdata/examples/main.wasm", // From machines testdata
		"machines/extism/testdata/examples/main.wasm",          // From project root to testdata
	}

	// Log the searched paths if logger is available
	if logger != nil {
		logger.Info("Searching for WASM file in multiple locations")
	}

	// Track checked paths for better error reporting
	checkedPaths := []string{}

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
		// Store the absolute path for error reporting
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path // Fallback to relative path if absolute path fails
		}
		checkedPaths = append(checkedPaths, absPath)
	}

	// If no WASM file found, provide detailed error with checked paths
	errMsg := `WASM file not found in any of the expected locations.

To fix this issue:
1. Run 'make build' in the machines/extism/testdata directory to generate the WASM file
2. OR copy a pre-built WASM file to one of these locations:
`

	for _, path := range checkedPaths {
		errMsg += "   - " + path + "\n"
	}

	return "", fmt.Errorf("%s", errMsg)
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

	// Find the WASM file
	wasmFilePath, err := findWasmFile(logger)
	if err != nil {
		return fmt.Errorf("failed to find WASM file: %w", err)
	}

	// Create evaluator with static and dynamic data providers
	evaluator, err := createExtismEvaluator(logger, wasmFilePath, staticData)
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
