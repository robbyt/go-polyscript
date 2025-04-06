package polyscript

import (
	"fmt"
	"log/slog"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/engine/options"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/machines"
	"github.com/robbyt/go-polyscript/machines/extism"
	"github.com/robbyt/go-polyscript/machines/risor"
	"github.com/robbyt/go-polyscript/machines/starlark"
	"github.com/robbyt/go-polyscript/machines/types"
)

// FromExtismFile creates an Extism evaluator from a WASM file
func FromExtismFile(filePath string, opts ...options.Option) (engine.EvaluatorWithPrep, error) {
	return fromFileLoader(types.Extism, filePath, opts...)
}

// FromRisorFile creates a Risor evaluator from a .risor file
func FromRisorFile(filePath string, opts ...options.Option) (engine.EvaluatorWithPrep, error) {
	return fromFileLoader(types.Risor, filePath, opts...)
}

// FromStarlarkFile creates a Starlark evaluator from a .star file
func FromStarlarkFile(filePath string, opts ...options.Option) (engine.EvaluatorWithPrep, error) {
	return fromFileLoader(types.Starlark, filePath, opts...)
}

// FromRisorString creates a Risor evaluator from a script string
func FromRisorString(content string, opts ...options.Option) (engine.EvaluatorWithPrep, error) {
	return fromStringLoader(types.Risor, content, opts...)
}

// FromStarlarkString creates a Starlark evaluator from a script string
func FromStarlarkString(content string, opts ...options.Option) (engine.EvaluatorWithPrep, error) {
	return fromStringLoader(types.Starlark, content, opts...)
}

// FromExtismFileWithData creates an Extism evaluator with both static and dynamic data capabilities.
// To add runtime data, use the `PrepareContext` method on the evaluator to add data to the context.
//
// Input parameters:
// - filePath: path to the WASM file
// - staticData: map of initial static data to be passed to the WASM module
// - logHandler: logger handler for logging
// - entryPoint: entry point for the WASM module (which function to call in the WASM file)
func FromExtismFileWithData(
	filePath string,
	staticData map[string]any,
	logHandler slog.Handler,
	entryPoint string,
) (engine.EvaluatorWithPrep, error) {
	return FromExtismFile(
		filePath,
		options.WithDefaults(),
		options.WithLogHandler(logHandler),
		withCompositeProvider(staticData),
		extism.WithEntryPoint(entryPoint),
	)
}

// FromRisorFileWithData creates an Risor evaluator with both static and dynamic data capabilities.
// To add runtime data, use the `PrepareContext` method on the evaluator to add data to the context.
//
// Input parameters:
// - filePath: path to the .risor script file
// - staticData: map of initial static data to be passed to the script
// - logHandler: logger handler for logging
func FromRisorFileWithData(
	filePath string,
	staticData map[string]any,
	logHandler slog.Handler,
) (engine.EvaluatorWithPrep, error) {
	return FromRisorFile(
		filePath,
		options.WithDefaults(),
		options.WithLogHandler(logHandler),
		withCompositeProvider(staticData),
	)
}

// FromStarlarkFileWithData creates an Risor evaluator with both static and dynamic data capabilities.
// To add runtime data, use the `PrepareContext` method on the evaluator to add data to the context.
//
// Input parameters:
// - filePath: path to the .risor script file
// - staticData: map of initial static data to be passed to the script
// - logHandler: logger handler for logging
func FromStarlarkFileWithData(
	filePath string,
	staticData map[string]any,
	logHandler slog.Handler,
) (engine.EvaluatorWithPrep, error) {
	return FromStarlarkFile(
		filePath,
		options.WithDefaults(),
		options.WithLogHandler(logHandler),
		withCompositeProvider(staticData),
	)
}

// FromRisorStringWithData creates a Risor evaluator with both static and dynamic data capabilities
// To add runtime data, use the `PrepareContext` method on the evaluator to add data to the context.
//
// Input parameters:
// - script: the Risor script as a string
// - staticData: map of initial static data to be passed to the script
// - logHandler: logger handler for logging
func FromRisorStringWithData(
	script string,
	staticData map[string]any,
	logHandler slog.Handler,
) (engine.EvaluatorWithPrep, error) {
	return FromRisorString(
		script,
		options.WithDefaults(),
		options.WithLogHandler(logHandler),
		withCompositeProvider(staticData),
		risor.WithGlobals([]string{constants.Ctx}),
	)
}

// FromStarlarkStringWithData creates a Starlark evaluator with both static and dynamic data
// capabilities. To add runtime data, use the `PrepareContext` method on the evaluator to add data
// to the context.
//
// Input parameters:
// - script: the Starlark script as a string
// - staticData: map of initial static data to be passed to the script
// - logHandler: logger handler for logging
func FromStarlarkStringWithData(
	script string,
	staticData map[string]any,
	logHandler slog.Handler,
) (engine.EvaluatorWithPrep, error) {
	return FromStarlarkString(
		script,
		options.WithDefaults(),
		options.WithLogHandler(logHandler),
		withCompositeProvider(staticData),
		starlark.WithGlobals([]string{constants.Ctx}),
	)
}

// NewExtismEvaluator creates a new evaluator for Extism WASM
func NewExtismEvaluator(opts ...options.Option) (engine.EvaluatorWithPrep, error) {
	return newEvaluator(types.Extism, opts...)
}

// NewStarlarkEvaluator creates a new evaluator for Starlark scripts
func NewStarlarkEvaluator(opts ...options.Option) (engine.EvaluatorWithPrep, error) {
	return newEvaluator(types.Starlark, opts...)
}

// NewRisorEvaluator creates a new evaluator for Risor scripts
func NewRisorEvaluator(opts ...options.Option) (engine.EvaluatorWithPrep, error) {
	return newEvaluator(types.Risor, opts...)
}

// withCompositeProvider creates an option to use a composite provider combining static and
// dynamic data. This allows adding some initial static data, as well as dynamic runtime data.
func withCompositeProvider(staticData map[string]any) options.Option {
	return func(cfg *options.Config) error {
		staticProvider := data.NewStaticProvider(staticData)
		dynamicProvider := data.NewContextProvider(constants.EvalData)
		compositeProvider := data.NewCompositeProvider(staticProvider, dynamicProvider)
		return options.WithDataProvider(compositeProvider)(cfg)
	}
}

// fromFileLoader creates an evaluator from a file path using the specified machine type
func fromFileLoader(
	machineType types.Type,
	filePath string,
	opts ...options.Option,
) (engine.EvaluatorWithPrep, error) {
	// Create a file loader
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	// Combine options, adding the loader
	allOpts := append([]options.Option{options.WithLoader(l)}, opts...)

	return newEvaluator(machineType, allOpts...)
}

// fromStringLoader creates an evaluator from a string content using the specified machine type
func fromStringLoader(
	machineType types.Type,
	content string,
	opts ...options.Option,
) (engine.EvaluatorWithPrep, error) {
	if machineType == types.Extism {
		return nil, fmt.Errorf("extism does not currently support string loaders")
	}

	// Create a string loader
	l, err := loader.NewFromString(content)
	if err != nil {
		return nil, err
	}

	// Combine options, adding the loader
	allOpts := append([]options.Option{options.WithLoader(l)}, opts...)

	return newEvaluator(machineType, allOpts...)
}

// newEvaluator creates a new evaluator for the specified machine type
func newEvaluator(
	machineType types.Type,
	opts ...options.Option,
) (engine.EvaluatorWithPrep, error) {
	// Initialize with machine-specific defaults
	cfg := options.DefaultConfig(machineType)

	// Apply all options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("error applying option: %w", err)
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return createEvaluator(cfg)
}

// createEvaluator is a helper function to create an evaluator from a config
func createEvaluator(cfg *options.Config) (engine.EvaluatorWithPrep, error) {
	// Create executable unit ID from source URL
	execUnitID := ""
	sourceURL := cfg.GetLoader().GetSourceURL()
	if sourceURL != nil {
		execUnitID = sourceURL.String()
	}

	// Create compiler
	compiler, err := machines.NewCompiler(
		cfg.GetHandler(),
		cfg.GetMachineType(),
		cfg.GetCompilerOptions(),
	)
	if err != nil {
		return nil, err
	}

	// Create executable unit (to compile and prepare the script)
	execUnit, err := script.NewExecutableUnit(
		cfg.GetHandler(),
		execUnitID,
		cfg.GetLoader(),
		compiler,
		cfg.GetDataProvider(),
	)
	if err != nil {
		return nil, err
	}

	// Create the machine-specific evaluator
	machineEvaluator, err := machines.NewEvaluator(cfg.GetHandler(), execUnit)
	if err != nil {
		return nil, err
	}

	// Wrap the evaluator to store the executable unit
	return NewEvaluatorWrapper(machineEvaluator, execUnit), nil
}
