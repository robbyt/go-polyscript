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

// RisorEvaluator is a type alias to make testing cleaner
type RisorEvaluator = engine.EvaluatorWithPrep

// GetRisorScript returns the script content for the Risor example
func GetRisorScript() string {
	return `
		// Wrap everything in a function for Risor syntax
		func process() {
			// Script has access to ctx variable passed from Go
			name := ctx["name"]
			timestamp := ctx["timestamp"]
			
			// Simple processing for tests
			result := {}
			result["greeting"] = "Hello, " + name + "!"
			result["timestamp"] = timestamp
			result["message"] = "Processed at " + timestamp
			
			return result
		}
		
		// Call the function and return its result
		process()
	`
}

// CreateRisorEvaluator creates a new Risor evaluator with the given script and logger
func CreateRisorEvaluator(scriptContent string, handler slog.Handler) (*RisorEvaluator, error) {
	// Define globals that will be available to the script
	globals := []string{constants.Ctx}

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
		return nil, err
	}

	return &evaluator, nil
}

// PrepareRequestData prepares the context with web request data
func PrepareRequestData(
	ctx context.Context,
	evaluator RisorEvaluator,
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

// EvaluateScript evaluates the script with the prepared context
func EvaluateScript(
	ctx context.Context,
	evaluator RisorEvaluator,
	logger *slog.Logger,
) (map[string]any, error) {
	logger.Info("Evaluating script")

	// Evaluate the script with the prepared context
	result, err := evaluator.Eval(ctx)
	if err != nil {
		logger.Error("Script evaluation failed", "error", err)
		return nil, fmt.Errorf("script evaluation failed: %w", err)
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

// DemonstrateDataPrepAndEval shows data preparation and evaluation as separate steps
func DemonstrateDataPrepAndEval(
	evaluator RisorEvaluator,
	logger *slog.Logger,
) (map[string]any, error) {
	// Create a base context
	ctx := context.Background()

	// Step 1: Prepare the context with request data
	logger.Info("Step 1: Preparing context with request data")
	enrichedCtx, err := PrepareRequestData(ctx, evaluator, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare context: %w", err)
	}

	// Step 2: Evaluate the script with the prepared context
	logger.Info("Step 2: Evaluating script with prepared context")
	result, err := EvaluateScript(enrichedCtx, evaluator, logger)
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
	logger := slog.New(handler.WithGroup("risor-data-prep-example"))

	// Create a script string
	scriptContent := GetRisorScript()

	// Create evaluator
	evaluator, err := CreateRisorEvaluator(scriptContent, handler)
	if err != nil {
		logger.Error("Failed to create evaluator", "error", err)
		os.Exit(1)
	}

	// Check if the evaluator is properly initialized
	if evaluator == nil {
		logger.Error("Evaluator is nil")
		os.Exit(1)
	}

	// Run the data preparation and evaluation example
	result, err := DemonstrateDataPrepAndEval(*evaluator, logger)
	if err != nil {
		logger.Error("Failed to demonstrate data prep and eval", "error", err)
		os.Exit(1)
	}

	// Print the result
	logger.Info("Final result", "data", result)
}
