package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/machines/risor"
)

func main() {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, nil)
	logger := slog.New(handler)

	// Define globals that will be available to the script
	globals := []string{constants.Ctx}

	// Create a compiler for Risor scripts
	compilerOptions := &risor.BasicCompilerOptions{Globals: globals}
	compiler := risor.NewCompiler(handler, compilerOptions)

	// Load script from a string
	scriptContent := `
		// Script has access to ctx variable passed from Go
		name := ctx["name"]
		message := "Hello, " + name + "!"
		
		// Return a map with our result
		{
			"greeting": message,
			"length": len(message)
		}
	`
	fromString, err := loader.NewFromString(scriptContent)
	if err != nil {
		logger.Error("Failed to create string loader", "error", err)
		return
	}

	// Create an executable unit
	unit, err := script.NewExecutableUnit(handler, "", fromString, compiler, nil)
	if err != nil {
		logger.Error("Failed to create executable unit", "error", err)
		return
	}

	// Create an evaluator for Risor scripts
	evaluator := risor.NewBytecodeEvaluator(handler)

	// Create context with input data
	ctx := context.Background()
	input := map[string]any{
		"name": "World",
	}
	ctx = context.WithValue(ctx, constants.EvalData, input)

	// Execute the script
	result, err := evaluator.Eval(ctx, unit)
	if err != nil {
		logger.Error("Script evaluation failed", "error", err)
		return
	}

	// Use the result
	fmt.Printf("Result: %v\n", result.Interface())
}
