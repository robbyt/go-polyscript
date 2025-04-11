package starlark

import (
	"fmt"
	"log/slog"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/machines/starlark/compiler"
	"github.com/robbyt/go-polyscript/machines/starlark/evaluator"
)

// FromStarlarkLoader creates a Starlark evaluator from a loader with dynamic data only (ContextProvider)
//
// Input parameters:
// - logHandler: logger handler for logging
// - ldr: loader implementation for loading the Starlark script content
//
// Returns an evaluator, which implements the engine.EvaluatorWithPrep interface.
func FromStarlarkLoader(
	logHandler slog.Handler,
	ldr loader.Loader,
) (*evaluator.BytecodeEvaluator, error) {
	return NewEvaluator(
		logHandler,
		ldr,
		data.NewContextProvider(constants.EvalData),
	)
}

// FromStarlarkLoaderWithData creates a Starlark evaluator with both static and dynamic data capabilities.
// To add runtime data, use the `PrepareContext` method on the evaluator to add data to the context.
//
// Input parameters:
// - logHandler: logger handler for logging
// - ldr: loader implementation for loading the Starlark script content
// - staticData: map of initial static data to be passed to the script
//
// Returns an evaluator, which implements the engine.EvaluatorWithPrep interface.
func FromStarlarkLoaderWithData(
	logHandler slog.Handler,
	ldr loader.Loader,
	staticData map[string]any,
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
	)
}

// NewCompiler creates a new Starlark compiler using the functional options pattern.
// See the starlarkMachine package for available compiler options. Returns a compiler,
// which implements the script.Compiler interface.
func NewCompiler(opts ...compiler.FunctionalOption) (*compiler.Compiler, error) {
	return compiler.NewCompiler(opts...)
}

// NewEvaluator creates a full Starlark evaluator with bytecode loaded, and ready for execution.
// Returns a BytecodeEvaluator, which implements the engine.EvaluatorWithPrep interface.
func NewEvaluator(
	logHandler slog.Handler,
	ldr loader.Loader,
	dataProvider data.Provider,
) (*evaluator.BytecodeEvaluator, error) {
	// Create compiler with the context global option
	compiler, err := NewCompiler(compiler.WithGlobals([]string{constants.Ctx}))
	if err != nil {
		return nil, fmt.Errorf("failed to create Starlark compiler: %w", err)
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
