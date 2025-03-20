package risor

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

// RunRisorExample executes a Risor script and returns the result
func RunRisorExample(handler slog.Handler) (map[string]any, error) {
	if handler == nil {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}
	logger := slog.New(handler)

	// Define globals that will be available to the script
	globals := []string{constants.Ctx}

	// Create a script string
	scriptContent := GetRisorScript()

	// Create input data
	input := map[string]any{
		"name": "World",
	}
	dataProvider := data.NewStaticProvider(input)

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

	// Execute the script
	ctx := context.Background()
	if evaluator == nil {
		logger.Error("Evaluator is nil")
		return nil, fmt.Errorf("evaluator is nil")
	}
	result, err := evaluator.Eval(ctx, nil)
	if err != nil {
		logger.Error("Script evaluation failed", "error", err, "evaluator", fmt.Sprintf("%T", evaluator))
		return nil, err
	}

	return result.Interface().(map[string]any), nil
}

func main() {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, nil)

	// Run the example
	result, err := RunRisorExample(handler)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print the result
	fmt.Printf("Result: %v\n", result)
}
