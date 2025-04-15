package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/platform"
)

// StarlarkEvaluator is a type alias to make testing cleaner
type StarlarkEvaluator = platform.Evaluator

//go:embed testdata/script.star
var starlarkScript string

// createStarlarkEvaluator creates a new Starlark evaluator with the given script and logger.
// Uses the simplified interface that automatically sets up static and dynamic data providers.
func createStarlarkEvaluator(
	logger *slog.Logger,
	scriptContent string,
	staticData map[string]any,
) (StarlarkEvaluator, error) {
	// Create evaluator using the new simplified interface
	// This automatically sets up a composite provider with both static and dynamic data
	return polyscript.FromStarlarkStringWithData(
		scriptContent,
		staticData,
		logger.Handler(),
	)
}

// prepareRuntimeData adds dynamic runtime data to the context.
// Returns the enriched context or an error.
func prepareRuntimeData(
	ctx context.Context,
	logger *slog.Logger,
	evaluator StarlarkEvaluator,
) (context.Context, error) {
	logger.Info("Preparing runtime data")

	// Create an HTTP request object (will not make a real request!)
	reqURL, err := url.Parse("http://localhost:8080/api/users?limit=10&offset=0")
	if err != nil {
		logger.Error("Failed to parse URL", "error", err)
		return nil, err
	}

	httpReq := &http.Request{
		Method: "GET",
		URL:    reqURL,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"User-Agent":   []string{"Example Client/1.0"},
		},
		Host:       "localhost",
		RemoteAddr: "127.0.1:8080",
	}

	// Create user data
	userData := map[string]any{
		"id":          "user-123",
		"role":        "admin",
		"permissions": "read,write,execute",
	}

	// General request metadata
	requestMeta := map[string]any{
		"name":      "World",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"user_data": userData,
	}

	// Add the request metadata to the context using the data.Provider
	enrichedCtx, err := evaluator.AddDataToContext(
		ctx,
		map[string]any{"request": httpReq},
		requestMeta,
	)
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
	evaluator StarlarkEvaluator,
) (map[string]any, error) {
	logger.Info("Evaluating script")

	// Evaluate the script with the prepared context
	result, err := evaluator.Eval(ctx)
	if err != nil {
		logger.Error("Script evaluation failed", "error", err)
		return nil, err
	}

	// Process the result
	val := result.Interface()
	if val == nil {
		logger.Warn("Result is nil")
		return map[string]any{}, nil
	}

	data, ok := val.(map[string]any)
	if !ok {
		logger.Error("Result is not a map", "type", fmt.Sprintf("%T", val))
		return nil, fmt.Errorf("result is not a map: %T", val)
	}

	logger.Info("Script evaluated successfully")
	return data, nil
}

func run() error {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler.WithGroup("starlark-data-prep-example"))

	// Static data loaded into a data provider, sent to the evaluator along with runtime data
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
	}

	// Create evaluator with static and dynamic data providers
	evaluator, err := createStarlarkEvaluator(logger, starlarkScript, staticData)
	if err != nil {
		return fmt.Errorf("failed to create evaluator: %w", err)
	}

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
