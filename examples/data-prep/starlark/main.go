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

// GetStarlarkScript returns the script content for the Starlark example
func GetStarlarkScript() string {
	return `
# Script has access to ctx variable passed from Go
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

# In Starlark, the last expression's value becomes the script's return value
result
`
}

// simulateWebRequest creates fake data to simulate an incoming web request
func simulateWebRequest() map[string]any {
	return map[string]any{
		"name":      "World",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	}
}

// simulateUserData creates fake user data to demonstrate multi-source data preparation
func simulateUserData() map[string]any {
	return map[string]any{
		"id":   "user-123",
		"role": "admin",
		"permissions": []string{
			"read", "write", "execute",
		},
	}
}

// PrepareAndEvaluateInSteps demonstrates the complete workflow with separate
// preparation steps, showing how different systems might handle different parts.
func PrepareAndEvaluateInSteps(
	evaluator engine.EvaluatorWithPrep,
	logger *slog.Logger,
) (map[string]any, error) {
	// Create a base context
	ctx := context.Background()

	// Step 1: System A - API Gateway
	// ------------------------------
	// In a real-world scenario, this might be an API gateway or frontend server
	// that receives the initial request but doesn't do script evaluation
	logger.Info("System A: API Gateway receiving request")
	requestData := simulateWebRequest()
	logger.Info("System A: Request data", "data", requestData)

	// First preparation step - enrich with request data
	ctx1, err := evaluator.PrepareContext(ctx, requestData)
	if err != nil {
		logger.Error("System A: Failed to prepare context with request data", "error", err)
		return nil, err
	}
	logger.Info("System A: Context prepared with request data")

	// Step 2: System B - Auth Service
	// ------------------------------
	// In a real-world scenario, this might be an authentication service
	// that enriches the context with user data
	logger.Info("System B: Auth Service enriching context with user data")
	userData := simulateUserData()
	logger.Info("System B: User data", "data", userData)

	// Add user data under a specific key
	userDataMap := map[string]any{
		"user_data": userData,
	}

	// Second preparation step - enrich with user data
	ctx2, err := evaluator.PrepareContext(ctx1, userDataMap)
	if err != nil {
		logger.Error("System B: Failed to prepare context with user data", "error", err)
		return nil, err
	}
	logger.Info("System B: Context prepared with user data")

	// Step 3: System C - Script Executor
	// ---------------------------------
	// In a real-world scenario, this might be a dedicated script execution service
	logger.Info("System C: Script Executor evaluating script with prepared context")

	// Evaluate the script with the fully prepared context
	result, err := evaluator.Eval(ctx2)
	if err != nil {
		logger.Error("System C: Script evaluation failed", "error", err)
		return nil, err
	}
	logger.Info("System C: Script evaluated successfully")

	// Process the result
	val := result.Interface()
	if val == nil {
		logger.Warn("System C: Result is nil")
		return map[string]any{}, nil
	}

	data, ok := val.(map[string]any)
	if !ok {
		logger.Error("System C: Result is not a map", "type", fmt.Sprintf("%T", val))
		return nil, fmt.Errorf("result is not a map: %T", val)
	}

	return data, nil
}

func main() {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	// Define globals that will be available to the script
	globals := []string{constants.Ctx}

	// Create a script string
	scriptContent := GetStarlarkScript()

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
		logger.Error("Failed to create evaluator", "error", err)
		os.Exit(1)
	}

	// Run the multi-step preparation and evaluation
	result, err := PrepareAndEvaluateInSteps(evaluator, logger)
	if err != nil {
		logger.Error("Failed to prepare and evaluate in steps", "error", err)
		os.Exit(1)
	}

	// Print the result
	logger.Info("Final result", "data", result)
}
