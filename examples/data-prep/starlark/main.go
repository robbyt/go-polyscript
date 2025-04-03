package main

import (
	"context"
	"fmt"
	"log/slog"
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

// GetStarlarkScript returns the script content for the Starlark example
func GetStarlarkScript() string {
	return `
# Script has access to ctx variable passed from Go
def main():
    name = ctx["name"]
    timestamp = ctx["timestamp"]

    # Starlark doesn't have time functions built-in, so we'll just use the timestamp directly
    current_time = timestamp

    # Process user data
    if ctx.get("user_data") != None:
        user_role = ctx["user_data"].get("role", "guest")
        user_id = ctx["user_data"].get("id", "unknown")
    else:
        user_role = "guest"
        user_id = "unknown"

    # Construct result dictionary
    result = {
        "greeting": "Hello, " + name + "!",
        "timestamp": timestamp,
        "message": "Processed by " + user_role + " at " + current_time,
        "user_id": user_id,
    }
    
    return result

# Call the main function
main()
`
}

// CreateStarlarkEvaluator creates a new Starlark evaluator with the given script and logger
func CreateStarlarkEvaluator(
	scriptContent string,
	handler slog.Handler,
) (*StarlarkEvaluator, error) {
	// Define globals that will be available to the script
	globals := []string{constants.Ctx}

	// Create data provider
	dataProvider := data.NewContextProvider(constants.EvalData)

	// Create evaluator using the functional options pattern
	evaluator, err := polyscript.FromStarlarkString(
		scriptContent,
		options.WithDefaults(),
		options.WithLogger(handler),
		options.WithDataProvider(dataProvider),
		starlark.WithGlobals(globals),
	)
	if err != nil {
		return nil, err
	}

	return &evaluator, nil
}

// PrepareRequestData prepares the context with basic request data
func PrepareRequestData(
	ctx context.Context,
	evaluator StarlarkEvaluator,
	logger *slog.Logger,
) (context.Context, error) {
	// Simulate gathering data from a web request
	logger.Info("Preparing request data")
	requestData := map[string]any{
		"name":      "World",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	}

	// Enrich context with request data
	logger.Info("Adding request data to context", "data", requestData)
	enrichedCtx, err := evaluator.PrepareContext(ctx, requestData)
	if err != nil {
		logger.Error("Failed to prepare context with request data", "error", err)
		return nil, err
	}

	logger.Info("Request data prepared successfully")
	return enrichedCtx, nil
}

// PrepareUserData prepares the context with user-specific data
func PrepareUserData(
	ctx context.Context,
	evaluator StarlarkEvaluator,
	logger *slog.Logger,
) (context.Context, error) {
	// Simulate gathering user data
	logger.Info("Preparing user data")
	userData := map[string]any{
		"id":          "user-123",
		"role":        "admin",
		"permissions": "read,write,execute",
	}

	// Add user data under a specific key
	userDataMap := map[string]any{
		"user_data": userData,
	}

	// Enrich context with user data
	logger.Info("Adding user data to context", "data", userDataMap)
	enrichedCtx, err := evaluator.PrepareContext(ctx, userDataMap)
	if err != nil {
		logger.Error("Failed to prepare context with user data", "error", err)
		return nil, err
	}

	logger.Info("User data prepared successfully")
	return enrichedCtx, nil
}

// EvaluateScript evaluates the script with the prepared context
func EvaluateScript(
	ctx context.Context,
	evaluator StarlarkEvaluator,
	logger *slog.Logger,
) (map[string]any, error) {
	logger.Info("Evaluating script")

	// Evaluate the script with the fully prepared context
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

// DemonstrateMultiStepPreparation shows the complete workflow with multiple preparation steps
func DemonstrateMultiStepPreparation(
	evaluator StarlarkEvaluator,
	logger *slog.Logger,
) (map[string]any, error) {
	// Create a base context
	ctx := context.Background()

	// Step 1: Prepare context with request data
	logger.Info("Step 1: Preparing context with request data")
	ctx1, err := PrepareRequestData(ctx, evaluator, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare context with request data: %w", err)
	}

	// Step 2: Prepare context with user data
	logger.Info("Step 2: Preparing context with user data")
	ctx2, err := PrepareUserData(ctx1, evaluator, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare context with user data: %w", err)
	}

	// Step 3: Evaluate the script
	logger.Info("Step 3: Evaluating script with fully prepared context")
	result, err := EvaluateScript(ctx2, evaluator, logger)
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
	logger := slog.New(handler.WithGroup("starlark-data-prep-example"))

	// Create a script string
	scriptContent := GetStarlarkScript()

	// Create evaluator
	evaluator, err := CreateStarlarkEvaluator(scriptContent, handler)
	if err != nil {
		logger.Error("Failed to create evaluator", "error", err)
		os.Exit(1)
	}

	// Run the multi-step preparation and evaluation
	result, err := DemonstrateMultiStepPreparation(*evaluator, logger)
	if err != nil {
		logger.Error("Failed to demonstrate multi-step preparation", "error", err)
		os.Exit(1)
	}

	// Print the result
	logger.Info("Final result", "data", result)
}
