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
// 3. Engine Implementations:
//   - Risor: Go-oriented scripting language
//   - Starlark: Python-like configuration language
//   - Extism: WebAssembly module execution (not benchmarked by default)
//
// To run these benchmarks, use the benchmark.sh script:
//
//	./benchmark.sh [pattern] [iterations]
//
// Example: ./benchmark.sh BenchmarkEvaluationPatterns 20x
package benchmarks

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/platform/constants"
)

// quietHandler is a slog.Handler that discards all logs
var quietHandler = slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})

// BenchmarkEvaluationPatterns compares different evaluation patterns:
// - Single execution (compile and run once)
// - Compile once, run many times
func BenchmarkEvaluationPatterns(b *testing.B) {
	// Simple script for benchmarking
	scriptContent := `
		let name = ctx["name"]
		let message = "Hello, " + name + "!"

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

			// Create and evaluate in each iteration (simulating one-time use)
			evaluator, err := polyscript.New[polyscript.Risor](
				b.Context(),
				polyscript.FromString(scriptContent),
				polyscript.WithStaticData[polyscript.Risor](inputData),
				polyscript.WithLogHandler[polyscript.Risor](quietHandler),
			)
			if err != nil {
				b.Fatalf("Failed to create evaluator: %v", err)
			}

			// Execute the script
			_, err = evaluator.Eval(b.Context())
			if err != nil {
				b.Fatalf("Failed to evaluate script: %v", err)
			}
		}
	})

	b.Run("CompileOnceRunMany", func(b *testing.B) {
		// Create evaluator once, outside the loop
		inputData := map[string]any{
			"name": "World",
		}
		evaluator, err := polyscript.New[polyscript.Risor](
			b.Context(),
			polyscript.FromString(scriptContent),
			polyscript.WithStaticData[polyscript.Risor](inputData),
			polyscript.WithLogHandler[polyscript.Risor](quietHandler),
		)
		if err != nil {
			b.Fatalf("Failed to create evaluator: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Just run evaluation in the loop
			_, err = evaluator.Eval(b.Context())
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
		let name = ctx["name"]
		let message = "Hello, " + name + "!"

		{
			"greeting": message,
			"length": len(message)
		}
	`

	inputData := map[string]any{
		"name": "World",
	}

	b.Run("StaticProvider", func(b *testing.B) {
		// WithStaticData sets up a StaticProvider
		evaluator, err := polyscript.New[polyscript.Risor](
			b.Context(),
			polyscript.FromString(scriptContent),
			polyscript.WithStaticData[polyscript.Risor](inputData),
			polyscript.WithLogHandler[polyscript.Risor](quietHandler),
		)
		if err != nil {
			b.Fatalf("Failed to create evaluator: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = evaluator.Eval(b.Context())
			if err != nil {
				b.Fatalf("Failed to evaluate script: %v", err)
			}
		}
	})

	b.Run("ContextProvider", func(b *testing.B) {
		// Without WithStaticData the evaluator uses a ContextProvider
		evaluator, err := polyscript.New[polyscript.Risor](
			b.Context(),
			polyscript.FromString(scriptContent),
			polyscript.WithLogHandler[polyscript.Risor](quietHandler),
		)
		if err != nil {
			b.Fatalf("Failed to create evaluator: %v", err)
		}

		ctx := context.WithValue(b.Context(), constants.EvalData, inputData)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = evaluator.Eval(ctx)
			if err != nil {
				b.Fatalf("Failed to evaluate script: %v", err)
			}
		}
	})

	b.Run("CompositeProvider", func(b *testing.B) {
		// For CompositeProvider use case, we can prepare the context separately
		// We access the name directly from the context
		compositeScript := `
			let name = ctx["name"]
			let message = "Hello, " + name + "!"
			{
				"greeting": message,
				"length": len(message)
			}
		`
		staticData := map[string]any{"defaultKey": "value"}
		evaluator, err := polyscript.New[polyscript.Risor](
			b.Context(),
			polyscript.FromString(compositeScript),
			polyscript.WithStaticData[polyscript.Risor](staticData), // Static part
			polyscript.WithLogHandler[polyscript.Risor](quietHandler),
		)
		if err != nil {
			b.Fatalf("Failed to create evaluator: %v", err)
		}

		ctx := b.Context()
		// Use AddDataToContext to add the dynamic part
		ctx, err = evaluator.AddDataToContext(ctx, map[string]any{"name": "World"})
		if err != nil {
			b.Fatalf("Failed to prepare context: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = evaluator.Eval(ctx)
			if err != nil {
				b.Fatalf("Failed to evaluate script: %v", err)
			}
		}
	})
}

// BenchmarkEngineComparison compares performance across different engine implementations:
// - Risor
// - Starlark
// - Extism
func BenchmarkEngineComparison(b *testing.B) {
	// Input data for all engines
	inputData := map[string]any{
		"name": "World",
	}

	// Risor script
	risorScript := `
		let name = ctx["name"]
		let message = "Hello, " + name + "!"

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

	b.Run("RisorEngine", func(b *testing.B) {
		evaluator, err := polyscript.New[polyscript.Risor](
			b.Context(),
			polyscript.FromString(risorScript),
			polyscript.WithStaticData[polyscript.Risor](inputData),
			polyscript.WithLogHandler[polyscript.Risor](quietHandler),
		)
		if err != nil {
			b.Fatalf("Failed to create Risor evaluator: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = evaluator.Eval(b.Context())
			if err != nil {
				b.Fatalf("Failed to evaluate Risor script: %v", err)
			}
		}
	})

	b.Run("StarlarkEngine", func(b *testing.B) {
		evaluator, err := polyscript.New[polyscript.Starlark](
			b.Context(),
			polyscript.FromString(starlarkScript),
			polyscript.WithStaticData[polyscript.Starlark](inputData),
			polyscript.WithLogHandler[polyscript.Starlark](quietHandler),
		)
		if err != nil {
			b.Fatalf("Failed to create Starlark evaluator: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = evaluator.Eval(b.Context())
			if err != nil {
				b.Fatalf("Failed to evaluate Starlark script: %v", err)
			}
		}
	})

	// Extism benchmark would be added here once we have a standard WASM file for testing
}
