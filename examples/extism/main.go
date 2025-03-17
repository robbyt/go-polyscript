package extism

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/machines"
	machineTypes "github.com/robbyt/go-polyscript/machines/types"
)

// FindWasmFile searches for the Extism WASM file in various likely locations
func FindWasmFile(logger *slog.Logger) (string, error) {
	paths := []string{
		"main.wasm",                 // Current directory
		"extism/main.wasm",          // extism subdirectory
		"examples/extism/main.wasm", // examples/extism subdirectory
		"../machines/extism/testdata/examples/main.wasm", // From machines testdata
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			absPath, err := filepath.Abs(path)
			if err == nil {
				if logger != nil {
					logger.Info("Found WASM file", "path", absPath)
				}
				return absPath, nil
			}
		}
	}

	return "", fmt.Errorf("WASM file not found in any of the expected locations")
}

// SimpleOptions implements the extism.CompilerOptions interface
type SimpleOptions struct {
	entryPoint string
}

func (s SimpleOptions) GetEntryPointName() string {
	return s.entryPoint
}

// RunExtismExample executes an Extism WASM module and returns the result
func RunExtismExample(handler slog.Handler) (map[string]any, error) {
	if handler == nil {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	logger := slog.New(handler.WithGroup("extism-example"))

	// Find the WASM file
	wasmFilePath, err := FindWasmFile(logger)
	if err != nil {
		logger.Error("Failed to find WASM file", "error", err)
		return nil, err
	}

	// Create a compiler for Extism scripts
	compilerOptions := SimpleOptions{entryPoint: "greet"}
	compiler, err := machines.NewCompiler(handler,
		machineTypes.Extism,
		compilerOptions)
	if err != nil {
		logger.Error("Failed to create compiler", "error", err)
		return nil, err
	}

	// Load the script from disk
	scriptContent, err := loader.NewFromDisk(wasmFilePath)
	if err != nil {
		logger.Error("Failed to load script", "error", err)
		return nil, err
	}

	// Create an executable unit
	executableUnit, err := script.NewExecutableUnit(handler, "", scriptContent, compiler, nil)
	if err != nil {
		logger.Error("Failed to create executable unit", "error", err)
		return nil, err
	}

	// Create an evaluator
	evaluator, err := machines.NewEvaluator(handler, executableUnit)
	if err != nil {
		logger.Error("Failed to create evaluator", "error", err)
		return nil, err
	}

	// Create a context with input data
	ctx := context.Background()
	inputData := map[string]any{
		"input": "World",
	}
	ctx = context.WithValue(ctx, constants.EvalData, inputData)

	// Set a timeout for script execution
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Evaluate the script
	response, err := evaluator.Eval(ctx, executableUnit)
	if err != nil {
		logger.Error("Failed to evaluate script", "error", err)
		return nil, err
	}

	// Return the result
	return response.Interface().(map[string]any), nil
}

func main() {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler.WithGroup("main"))

	// Run the example
	result, err := RunExtismExample(handler)
	if err != nil {
		logger.Error("Failed to run Extism example", "error", err)
		os.Exit(1)
	}

	// Print the result
	logger.Info("Result", "data", result)
}
