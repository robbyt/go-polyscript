package extism

import (
	"fmt"
	"log/slog"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/machines/extism/compiler"
	"github.com/robbyt/go-polyscript/machines/extism/evaluator"
)

// FromExtismLoader creates an Extism evaluator from a loader with dynamic data only (ContextProvider)
//
// Input parameters:
// - l: loader implementation for loading the WASM content
// - logHandler: logger handler for logging
// - entryPoint: entry point for the WASM module (which function to call in the WASM file)
//
// Returns an evaluator, which implements the engine.EvaluatorWithPrep interface.
func FromExtismLoader(
	logHandler slog.Handler,
	ldr loader.Loader,
	entryPoint string,
) (*evaluator.BytecodeEvaluator, error) {
	return NewEvaluator(
		logHandler,
		ldr,
		data.NewContextProvider(constants.EvalData),
		entryPoint,
	)
}

// FromExtismLoaderWithData creates an Extism evaluator with both static and dynamic data capabilities.
// To add runtime data, use the `PrepareContext` method on the evaluator to add data to the context.
//
// Input parameters:
// - l: loader implementation for loading the WASM content
// - staticData: map of initial static data to be passed to the WASM module
// - logHandler: logger handler for logging
// - entryPoint: entry point for the WASM module (which function to call in the WASM file)
//
// Returns an evaluator, which implements the engine.EvaluatorWithPrep interface.
func FromExtismLoaderWithData(
	logHandler slog.Handler,
	ldr loader.Loader,
	staticData map[string]any,
	entryPoint string,
) (*evaluator.BytecodeEvaluator, error) {
	// Create a composite provider with the static, and dynamic data loader
	staticProvider := data.NewStaticProvider(staticData)
	dynamicProvider := data.NewContextProvider(constants.EvalData)
	compositeProvider := data.NewCompositeProvider(staticProvider, dynamicProvider)

	// Create the evaluator
	return NewEvaluator(
		logHandler,
		ldr,
		compositeProvider,
		entryPoint,
	)
}

// NewCompiler creates a new Extism compiler using the functional options pattern.
// See the extismMachine package for available compiler options. Returns a compiler,
// which implements the script.Compiler interface.
func NewCompiler(opts ...compiler.FunctionalOption) (*compiler.Compiler, error) {
	return compiler.NewCompiler(opts...)
}

// NewEvaluator creates an full Extism evaluator with bytecode loaded, and ready for execution.
// Returns a BytecodeEvaluator, which implements the engine.EvaluatorWithPrep interface.
func NewEvaluator(
	logHandler slog.Handler,
	ldr loader.Loader,
	dataProvider data.Provider,
	entryPoint string,
) (*evaluator.BytecodeEvaluator, error) {
	// Create compiler with the entry point option
	compiler, err := NewCompiler(compiler.WithEntryPoint(entryPoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create Extism compiler: %w", err)
	}

	// Create executable unit ID from source URL
	execUnitID := ""
	sourceURL := ldr.GetSourceURL()
	if sourceURL != nil {
		execUnitID = sourceURL.String()
	}

	// Create executable unit (to compile and prepare the script)
	execUnit, err := script.NewExecutableUnit(
		logHandler,
		execUnitID,
		ldr,
		compiler,
		dataProvider,
	)
	if err != nil {
		return nil, err
	}

	// BytecodeEvaluator already implements the EvaluatorWithPrep interface
	return evaluator.NewBytecodeEvaluator(logHandler, execUnit), nil
}
