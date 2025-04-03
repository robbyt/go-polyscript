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
	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/starlark"
	"github.com/robbyt/go-polyscript/options"
)

// StarlarkEvaluator is a type alias to make testing cleaner
type StarlarkEvaluator = engine.EvaluatorWithPrep

//go:embed testdata/script.star
var starlarkScript string

// createStarlarkEvaluator creates a new Starlark evaluator with the given script and logger.
// Sets up a CompositeProvider that combines static and dynamic data providers.
func createStarlarkEvaluator(
	logger *slog.Logger,
	scriptContent string,
	staticData map[string]any,
) (StarlarkEvaluator, error) {
	// Define globals that will be available to the script
	globals := []string{constants.Ctx}

	// the static provider enables access to the static data map
	staticProvider := data.NewStaticProvider(staticData)

	// this context provider enables each request to add different dynamic data
	dynamicProvider := data.NewContextProvider(constants.EvalData)

	// Composite provider handles static data first, then dynamic data
	compositeProvider := data.NewCompositeProvider(staticProvider, dynamicProvider)

	// Create evaluator using the functional options pattern
	return polyscript.FromStarlarkString(
		scriptContent,
		options.WithDefaults(),
		options.WithLogger(logger.Handler()),
		options.WithDataProvider(compositeProvider),
		starlark.WithGlobals(globals),
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

	// Create an HTTP request object
	reqURL, err := url.Parse("https://example.com/api/users?limit=10&offset=0")
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
		Host:       "example.com",
		RemoteAddr: "192.168.1.1:12345",
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
	enrichedCtx, err := evaluator.PrepareContext(ctx, httpReq, requestMeta)
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
