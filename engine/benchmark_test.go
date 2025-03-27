// Package engine_test contains benchmarks for go-polyscript.
//
// This file contains benchmarks for measuring go-polyscript performance characteristics.
// The benchmarks compare different patterns and configurations:
//
// 1. Evaluation Patterns:
//   - SingleExecution: Creates a new evaluator for each execution (slower)
//   - CompileOnceRunMany: Reuses a compiled evaluator (faster)
//
// 2. Data Providers:
//   - StaticProvider: Fixed data provided at creation time
//   - ContextProvider: Data retrieved from context at runtime
//   - CompositeProvider: Combines multiple providers
//
// 3. VM Implementations:
//   - RisorVM: Go-oriented scripting language
//   - StarlarkVM: Python-like configuration language
//   - ExtismVM: WebAssembly module execution (not benchmarked by default)
//
// To run these benchmarks, use the benchmark.sh script:
//
//	./benchmark.sh [pattern] [iterations]
//
// Example: ./benchmark.sh BenchmarkEvaluationPatterns 20x
package engine_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/risor"
	"github.com/robbyt/go-polyscript/machines/starlark"
	"github.com/robbyt/go-polyscript/options"
)

// quietHandler is a slog.Handler that discards all logs
var quietHandler = slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})

// BenchmarkEvaluationPatterns compares different evaluation patterns:
// - Single execution (compile and run once)
// - Compile once, run many times
func BenchmarkEvaluationPatterns(b *testing.B) {
	// Simple script for benchmarking
	scriptContent := `
		name := ctx["name"]
		message := "Hello, " + name + "!"
		
		{
			"greeting": message,
			"length": len(message)
		}
	`

	b.Run("SingleExecution", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Create input data
			inputData := map[string]any{
				"name": "World",
			}
			dataProvider := data.NewStaticProvider(inputData)

			// Create and evaluate in each iteration (simulating one-time use)
			evaluator, err := polyscript.FromRisorString(
				scriptContent,
				options.WithDefaults(),
				options.WithDataProvider(dataProvider),
				options.WithLogger(quietHandler),
				risor.WithGlobals([]string{constants.Ctx}),
			)
			if err != nil {
				b.Fatalf("Failed to create evaluator: %v", err)
			}

			// Execute the script
			_, err = evaluator.Eval(context.Background())
			if err != nil {
				b.Fatalf("Failed to evaluate script: %v", err)
			}
		}
	})

	b.Run("CompileOnceRunMany", func(b *testing.B) {
		// Create evaluator once, outside the loop
		dataProvider := data.NewStaticProvider(map[string]any{
			"name": "World",
		})
		evaluator, err := polyscript.FromRisorString(
			scriptContent,
			options.WithDefaults(),
			options.WithDataProvider(dataProvider),
			options.WithLogger(quietHandler),
			risor.WithGlobals([]string{constants.Ctx}),
		)
		if err != nil {
			b.Fatalf("Failed to create evaluator: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Just run evaluation in the loop
			_, err = evaluator.Eval(context.Background())
			if err != nil {
				b.Fatalf("Failed to evaluate script: %v", err)
			}
		}
	})
}

// BenchmarkDataProviders compares different data provider types:
// - StaticProvider
// - ContextProvider
// - CompositeProvider
func BenchmarkDataProviders(b *testing.B) {
	// Simple script for benchmarking
	scriptContent := `
		name := ctx["name"]
		message := "Hello, " + name + "!"
		
		{
			"greeting": message,
			"length": len(message)
		}
	`

	inputData := map[string]any{
		"name": "World",
	}

	b.Run("StaticProvider", func(b *testing.B) {
		dataProvider := data.NewStaticProvider(inputData)
		evaluator, err := polyscript.FromRisorString(
			scriptContent,
			options.WithDefaults(),
			options.WithDataProvider(dataProvider),
			options.WithLogger(quietHandler),
			risor.WithGlobals([]string{constants.Ctx}),
		)
		if err != nil {
			b.Fatalf("Failed to create evaluator: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = evaluator.Eval(context.Background())
			if err != nil {
				b.Fatalf("Failed to evaluate script: %v", err)
			}
		}
	})

	b.Run("ContextProvider", func(b *testing.B) {
		dataProvider := data.NewContextProvider(constants.EvalData)
		evaluator, err := polyscript.FromRisorString(
			scriptContent,
			options.WithDefaults(),
			options.WithDataProvider(dataProvider),
			options.WithLogger(quietHandler),
			risor.WithGlobals([]string{constants.Ctx}),
		)
		if err != nil {
			b.Fatalf("Failed to create evaluator: %v", err)
		}

		ctx := context.WithValue(context.Background(), constants.EvalData, inputData)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = evaluator.Eval(ctx)
			if err != nil {
				b.Fatalf("Failed to evaluate script: %v", err)
			}
		}
	})

	b.Run("CompositeProvider", func(b *testing.B) {
		staticProvider := data.NewStaticProvider(map[string]any{"defaultKey": "value"})
		contextProvider := data.NewContextProvider(constants.EvalData)
		compositeProvider := data.NewCompositeProvider(contextProvider, staticProvider)

		evaluator, err := polyscript.FromRisorString(
			scriptContent,
			options.WithDefaults(),
			options.WithDataProvider(compositeProvider),
			options.WithLogger(quietHandler),
			risor.WithGlobals([]string{constants.Ctx}),
		)
		if err != nil {
			b.Fatalf("Failed to create evaluator: %v", err)
		}

		ctx := context.WithValue(context.Background(), constants.EvalData, inputData)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = evaluator.Eval(ctx)
			if err != nil {
				b.Fatalf("Failed to evaluate script: %v", err)
			}
		}
	})
}

// BenchmarkVMComparison compares performance across different VM implementations:
// - Risor
// - Starlark
// - Extism
func BenchmarkVMComparison(b *testing.B) {
	// Input data for all VMs
	inputData := map[string]any{
		"name": "World",
	}
	staticProvider := data.NewStaticProvider(inputData)

	// Risor script
	risorScript := `
		name := ctx["name"]
		message := "Hello, " + name + "!"
		
		{
			"greeting": message,
			"length": len(message)
		}
	`

	// Starlark script - Note: Starlark is whitespace sensitive, so no indentation
	starlarkScript := `
name = ctx["name"]
message = "Hello, " + name + "!"

{"greeting": message, "length": len(message)}
`

	// Note: Extism benchmark would need an actual WASM file,
	// which is more complex to set up in this benchmark template

	b.Run("RisorVM", func(b *testing.B) {
		evaluator, err := polyscript.FromRisorString(
			risorScript,
			options.WithDefaults(),
			options.WithDataProvider(staticProvider),
			options.WithLogger(quietHandler),
			risor.WithGlobals([]string{constants.Ctx}),
		)
		if err != nil {
			b.Fatalf("Failed to create Risor evaluator: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = evaluator.Eval(context.Background())
			if err != nil {
				b.Fatalf("Failed to evaluate Risor script: %v", err)
			}
		}
	})

	b.Run("StarlarkVM", func(b *testing.B) {
		evaluator, err := polyscript.FromStarlarkString(
			starlarkScript,
			options.WithDefaults(),
			options.WithDataProvider(staticProvider),
			options.WithLogger(quietHandler),
			starlark.WithGlobals([]string{constants.Ctx}),
		)
		if err != nil {
			b.Fatalf("Failed to create Starlark evaluator: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = evaluator.Eval(context.Background())
			if err != nil {
				b.Fatalf("Failed to evaluate Starlark script: %v", err)
			}
		}
	})

	// Extism benchmark would be added here once we have a standard WASM file for testing
}
