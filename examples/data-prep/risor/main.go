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
	"github.com/robbyt/go-polyscript/machines/risor"
	"github.com/robbyt/go-polyscript/options"
)

// GetRisorScript returns the script content for the Risor example
func GetRisorScript() string {
	return `
		// Script has access to ctx variable passed from Go
		name := ctx["name"]
		timestamp := ctx["timestamp"]
		
		// Format the timestamp nicely
		formatted_time := time.Parse(timestamp, "2006-01-02 15:04:05")
		
		// Return a map with our result
		{
			"greeting": "Hello, " + name + "!",
			"timestamp": timestamp,
			"message": "Processed at " + formatted_time,
		}
	`
}

// simulateWebRequest creates fake data to simulate an incoming web request
func simulateWebRequest() map[string]any {
	return map[string]any{
		"name":      "World",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	}
}

// simulateDistributedArchitecture demonstrates how PrepareContext and Eval
// can be separated to enable a distributed architecture pattern.
//
// This pattern is useful when:
// 1. Data preparation and evaluation need to happen on different systems
// 2. Context enrichment is complex and should be separated from evaluation
// 3. You want to prepare data once and evaluate multiple times
func simulateDistributedArchitecture(evaluator engine.EvaluatorWithPrep, logger *slog.Logger) (map[string]any, error) {
	// Create a base context
	ctx := context.Background()

	// 1. Simulate Web Server: Prepare the context with request data
	// -------------------------------------------------------------
	logger.Info("Web Server: Preparing context with request data")
	requestData := simulateWebRequest()
	logger.Info("Web Server: Request data", "data", requestData)

	// Use the PrepareContext method to enrich the context with the request data
	// This could happen on a web server that receives the request
	enrichedCtx, err := evaluator.PrepareContext(ctx, requestData)
	if err != nil {
		logger.Error("Web Server: Failed to prepare context", "error", err)
		return nil, fmt.Errorf("failed to prepare context: %w", err)
	}
	logger.Info("Web Server: Context prepared successfully")

	// At this point, in a distributed architecture, the enriched context
	// would be serialized and sent to a worker for evaluation

	// 2. Simulate Worker: Evaluate the script with the prepared context
	// ----------------------------------------------------------------
	logger.Info("Worker: Receiving prepared context and evaluating script")

	// Evaluate the script with the prepared context
	// This could happen on a separate worker system
	result, err := evaluator.Eval(enrichedCtx)
	if err != nil {
		logger.Error("Worker: Script evaluation failed", "error", err)
		return nil, fmt.Errorf("script evaluation failed: %w", err)
	}
	logger.Info("Worker: Script evaluated successfully")

	// Process the result
	val := result.Interface()
	if val == nil {
		logger.Warn("Worker: Result is nil")
		return map[string]any{}, nil
	}

	data, ok := val.(map[string]any)
	if !ok {
		logger.Error("Worker: Result is not a map", "type", fmt.Sprintf("%T", val))
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
	scriptContent := GetRisorScript()

	// Create data provider
	dataProvider := data.NewContextProvider(constants.EvalData)

	// Create evaluator using the functional options pattern
	evaluator, err := polyscript.FromRisorString(
		scriptContent,
		options.WithDefaults(),
		options.WithLogger(handler),
		options.WithDataProvider(dataProvider),
		risor.WithGlobals(globals),
	)
	if err != nil {
		logger.Error("Failed to create evaluator", "error", err)
		os.Exit(1)
	}

	// Check if the evaluator is properly initialized
	if evaluator == nil {
		logger.Error("Evaluator is nil")
		os.Exit(1)
	}

	// Run the distributed architecture simulation
	result, err := simulateDistributedArchitecture(evaluator, logger)
	if err != nil {
		logger.Error("Failed to simulate distributed architecture", "error", err)
		os.Exit(1)
	}

	// Print the result
	logger.Info("Final result", "data", result)
}
