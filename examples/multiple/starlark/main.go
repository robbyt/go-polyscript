package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript"
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
message = "Hello, " + name + "!"

# Return a dictionary with our result - must explicitly return to get a value
result = {
    "greeting": message,
    "length": len(message)
}
# In Starlark, the last expression's value becomes the script's return value
result
`
}

// RunStarlarkExampleMultipleTimes demonstrates the "compile once, run many times" pattern
// It compiles the script once and then executes it multiple times with different inputs.
//
// This pattern is more efficient than compiling the script for each execution:
// 1. It creates a script evaluator with a ContextProvider to get data from context
// 2. For each execution, it passes different data via the context
// 3. The ContextProvider retrieves this data during execution
// 4. The script accesses data through the "ctx" global variable
func RunStarlarkExampleMultipleTimes(handler slog.Handler) ([]map[string]any, error) {
	if handler == nil {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}
	logger := slog.New(handler)

	// Define globals that will be available to the script
	globals := []string{constants.Ctx}

	// Create a script string
	scriptContent := GetStarlarkScript()

	// Create a context provider that will be used for runtime data
	// This allows us to pass different data on each evaluation
	ctxProvider := data.NewContextProvider(constants.EvalData)

	// Create evaluator using the functional options pattern
	evaluator, err := polyscript.FromStarlarkString(
		scriptContent,
		options.WithDefaults(),
		options.WithLogger(handler),
		options.WithDataProvider(ctxProvider), // Use context provider
		starlark.WithGlobals(globals),
	)
	if err != nil {
		logger.Error("Failed to create evaluator", "error", err)
		return nil, err
	}

	// Names to greet - we'll run the script once for each name
	names := []string{"World", "Alice", "Bob", "Charlie"}
	results := make([]map[string]any, 0, len(names))

	// Execute the script multiple times with different input data
	for _, name := range names {
		// Create the data structure expected by the script
		// This is a simple map that will be made available to the script via the "ctx" global
		// The script can access values like: ctx["name"]
		contextData := map[string]any{
			"name": name,
		}

		// Create a context with the specific data for this run
		ctx := context.Background()
		// Important: Use the EvalData constant for context value key to match the ContextProvider
		ctx = context.WithValue(ctx, constants.EvalData, contextData)

		logger.Debug("Running script with name", "name", name)

		// Execute the script with the context
		result, err := evaluator.Eval(ctx)
		if err != nil {
			logger.Error("Script evaluation failed", "error", err, "name", name)
			continue
		}

		// Process result
		val := result.Interface()
		if val == nil {
			logger.Warn("Result is nil for name", "name", name)
			continue
		}

		data, ok := val.(map[string]any)
		if !ok {
			logger.Error("Result is not a map", "type", fmt.Sprintf("%T", val), "name", name)
			continue
		}

		results = append(results, data)
		logger.Info("Processed result", "name", name, "greeting", data["greeting"])
	}

	return results, nil
}

func main() {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, nil)

	// Run the example
	results, err := RunStarlarkExampleMultipleTimes(handler)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print the results
	for i, result := range results {
		fmt.Printf("Result %d: %v\n", i+1, result)
	}
}
