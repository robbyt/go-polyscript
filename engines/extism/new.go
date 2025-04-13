package extism

import (
	"fmt"
	"log/slog"

	"github.com/robbyt/go-polyscript/engines/extism/compiler"
	"github.com/robbyt/go-polyscript/engines/extism/evaluator"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

// FromExtismLoader creates an Extism evaluator from a loader with dynamic data only (ContextProvider)
//
// Input parameters:
// - l: loader implementation for loading the WASM content
// - logHandler: logger handler for logging
// - entryPoint: entry point for the WASM module (which function to call in the WASM file)
//
// Returns an evaluator, which implements the evaluation.Evaluator interface.
func FromExtismLoader(
	logHandler slog.Handler,
	ldr loader.Loader,
	entryPoint string,
) (*evaluator.Evaluator, error) {
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
// Returns an evaluator, which implements the evaluation.Evaluator interface.
func FromExtismLoaderWithData(
	logHandler slog.Handler,
	ldr loader.Loader,
	staticData map[string]any,
	entryPoint string,
) (*evaluator.Evaluator, error) {
	staticProvider := data.NewStaticProvider(staticData)
	dynamicProvider := data.NewContextProvider(constants.EvalData)
	compositeProvider := data.NewCompositeProvider(staticProvider, dynamicProvider)

	return NewEvaluator(
		logHandler,
		ldr,
		compositeProvider,
		entryPoint,
	)
}

// NewCompiler creates a new Extism compiler using the functional options pattern.
// Returns a compiler implementing the script.Compiler interface.
func NewCompiler(opts ...compiler.FunctionalOption) (*compiler.Compiler, error) {
	return compiler.New(opts...)
}

// NewEvaluator creates an Extism evaluator with WASM code loaded, and ready for execution.
// Returns a Evaluator, which implements the evaluation.Evaluator interface.
func NewEvaluator(
	logHandler slog.Handler,
	ldr loader.Loader,
	dataProvider data.Provider,
	entryPoint string,
) (*evaluator.Evaluator, error) {
	if dataProvider == nil {
		return nil, fmt.Errorf("provider is nil")
	}

	compiler, err := NewCompiler(compiler.WithEntryPoint(entryPoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create Extism compiler: %w", err)
	}

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

	return evaluator.New(logHandler, execUnit), nil
}
