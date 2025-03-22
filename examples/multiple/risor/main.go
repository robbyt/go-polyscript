package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript"
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
		message := "Hello, " + name + "!"
		
		// Return a map with our result
		{
			"greeting": message,
			"length": len(message)
		}
	`
}

// RunRisorExampleMultipleTimes demonstrates the "compile once, run many times" pattern
// It compiles the script once and then executes it multiple times with different inputs.
//
// This is much more efficient than compiling the script for each execution.
func RunRisorExampleMultipleTimes(handler slog.Handler) ([]map[string]any, error) {
	if handler == nil {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}
	logger := slog.New(handler)

	// Define globals that will be available to the script
	globals := []string{constants.Ctx}

	// Create a script string
	scriptContent := GetRisorScript()

	// Create a context provider - this allows us to inject different data on each evaluation
	dataProvider := data.NewContextProvider(constants.EvalData)

	// Create evaluator using the functional options pattern
	evaluator, err := polyscript.FromRisorString(
		scriptContent,
		options.WithDefaults(), // Add defaults option to ensure all required fields are set
		options.WithLogger(handler),
		options.WithDataProvider(dataProvider),
		risor.WithGlobals(globals),
	)
	if err != nil {
		logger.Error("Failed to create evaluator", "error", err)
		return nil, err
	}

	// Names to greet - we'll run the script once for each name
	names := []string{"World", "Risor", "Go"}
	results := make([]map[string]any, 0, len(names))

	// Execute the script multiple times with different inputs
	for i, name := range names {
		// Create the data structure expected by the script
		// This is a simple map that will be made available to the script via the "ctx" global
		contextData := map[string]any{
			"name": name,
		}

		// Create a context with the specific data for this run
		ctx := context.Background()
		// Important: Use the EvalData constant for context value key to match the ContextProvider
		ctx = context.WithValue(ctx, constants.EvalData, contextData)

		// Execute the script with the context
		result, err := evaluator.Eval(ctx)
		if err != nil {
			logger.Error("Script evaluation failed",
				"error", err,
				"evaluator", fmt.Sprintf("%T", evaluator),
				"execution", i+1)
			return nil, err
		}

		// Get the result as a map
		val := result.Interface()
		if val == nil {
			logger.Warn("Result is nil", "execution", i+1)
			continue
		}

		data, ok := val.(map[string]any)
		if !ok {
			logger.Error("Result is not a map",
				"type", fmt.Sprintf("%T", val),
				"execution", i+1)
			return nil, fmt.Errorf("result is not a map: %T", val)
		}

		// Add the result to the list
		results = append(results, data)
	}

	return results, nil
}

func main() {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, nil)

	// Run the example
	results, err := RunRisorExampleMultipleTimes(handler)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print the results
	for i, result := range results {
		fmt.Printf("Result %d: %v\n", i+1, result)
	}
}
