package risor

import (
	"fmt"
	"log/slog"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/machines/risor/compiler"
	"github.com/robbyt/go-polyscript/machines/risor/evaluator"
)

// FromRisorLoader creates a Risor evaluator from a loader with dynamic data only (ContextProvider)
//
// Input parameters:
// - logHandler: logger handler for logging
// - ldr: loader implementation for loading the Risor script content
//
// Returns an evaluator, which implements the engine.EvaluatorWithPrep interface.
func FromRisorLoader(
	logHandler slog.Handler,
	ldr loader.Loader,
) (*evaluator.Evaluator, error) {
	return NewEvaluator(
		logHandler,
		ldr,
		data.NewContextProvider(constants.EvalData),
	)
}

// FromRisorLoaderWithData creates a Risor evaluator with both static and dynamic data capabilities.
// To add runtime data, use the `PrepareContext` method on the evaluator to add data to the context.
//
// Input parameters:
// - logHandler: logger handler for logging
// - ldr: loader implementation for loading the Risor script content
// - staticData: map of initial static data to be passed to the script
//
// Returns an evaluator, which implements the engine.EvaluatorWithPrep interface.
func FromRisorLoaderWithData(
	logHandler slog.Handler,
	ldr loader.Loader,
	staticData map[string]any,
) (*evaluator.Evaluator, error) {
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

// NewCompiler creates a new Risor compiler using the functional options pattern.
// Returns a compiler implementing the script.Compiler interface.
func NewCompiler(opts ...compiler.FunctionalOption) (*compiler.Compiler, error) {
	return compiler.New(opts...)
}

// NewEvaluator creates a full Risor evaluator with bytecode loaded, and ready for execution.
// Returns a Evaluator, which implements the engine.EvaluatorWithPrep interface.
func NewEvaluator(
	logHandler slog.Handler,
	ldr loader.Loader,
	dataProvider data.Provider,
) (*evaluator.Evaluator, error) {
	// Validate provider is not nil
	if dataProvider == nil {
		return nil, fmt.Errorf("provider is nil")
	}

	// Create compiler with the context global option
	compiler, err := NewCompiler(compiler.WithCtxGlobal())
	if err != nil {
		return nil, fmt.Errorf("failed to create Risor compiler: %w", err)
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

	return evaluator.New(logHandler, execUnit), nil
}
