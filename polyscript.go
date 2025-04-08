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
func FromExtismFile(filePath string, opts ...any) (engine.EvaluatorWithPrep, error) {
	return fromFileLoader(types.Extism, filePath, opts...)
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
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	// Create an evaluator with specific options
	cfg := options.DefaultConfig(types.Extism)
	cfg.SetHandler(logHandler)
	cfg.SetLoader(l)

	// Set the data provider
	cfg.SetDataProvider(buildCompositeDataProvider(staticData))

	// Create an evaluator using our custom createExtismEvaluator function
	return createExtismEvaluator(cfg, []extism.CompilerOption{extism.WithEntryPoint(entryPoint)})
}

// FromRisorFile creates a Risor evaluator from a .risor file
func FromRisorFile(filePath string, opts ...any) (engine.EvaluatorWithPrep, error) {
	return fromFileLoader(types.Risor, filePath, opts...)
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
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	// Create an evaluator with specific options
	cfg := options.DefaultConfig(types.Risor)
	cfg.SetHandler(logHandler)
	cfg.SetLoader(l)

	// Set the data provider
	cfg.SetDataProvider(buildCompositeDataProvider(staticData))

	// Create an evaluator using our custom createRisorEvaluator function
	return createRisorEvaluator(
		cfg,
		[]risor.CompilerOption{risor.WithGlobals([]string{constants.Ctx})},
	)
}

// FromRisorString creates a Risor evaluator from a script string
func FromRisorString(content string, opts ...any) (engine.EvaluatorWithPrep, error) {
	return fromStringLoader(types.Risor, content, opts...)
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
	l, err := loader.NewFromString(script)
	if err != nil {
		return nil, err
	}

	// Create an evaluator with specific options
	cfg := options.DefaultConfig(types.Risor)
	cfg.SetHandler(logHandler)
	cfg.SetLoader(l)

	// Set the data provider
	cfg.SetDataProvider(buildCompositeDataProvider(staticData))

	// Create an evaluator using our custom createRisorEvaluator function
	return createRisorEvaluator(
		cfg,
		[]risor.CompilerOption{risor.WithGlobals([]string{constants.Ctx})},
	)
}

// FromStarlarkFile creates a Starlark evaluator from a .star file
func FromStarlarkFile(filePath string, opts ...any) (engine.EvaluatorWithPrep, error) {
	return fromFileLoader(types.Starlark, filePath, opts...)
}

// FromStarlarkFileWithData creates a Starlark evaluator with both static and dynamic data capabilities.
// To add runtime data, use the `PrepareContext` method on the evaluator to add data to the context.
//
// Input parameters:
// - filePath: path to the .star script file
// - staticData: map of initial static data to be passed to the script
// - logHandler: logger handler for logging
func FromStarlarkFileWithData(
	filePath string,
	staticData map[string]any,
	logHandler slog.Handler,
) (engine.EvaluatorWithPrep, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	// Create an evaluator with specific options
	cfg := options.DefaultConfig(types.Starlark)
	cfg.SetHandler(logHandler)
	cfg.SetLoader(l)

	// Set the data provider
	cfg.SetDataProvider(buildCompositeDataProvider(staticData))

	// Create an evaluator using our custom createStarlarkEvaluator function
	return createStarlarkEvaluator(
		cfg,
		[]starlark.CompilerOption{starlark.WithGlobals([]string{constants.Ctx})},
	)
}

// FromStarlarkString creates a Starlark evaluator from a script string
func FromStarlarkString(content string, opts ...any) (engine.EvaluatorWithPrep, error) {
	return fromStringLoader(types.Starlark, content, opts...)
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
	l, err := loader.NewFromString(script)
	if err != nil {
		return nil, err
	}

	// Create an evaluator with specific options
	cfg := options.DefaultConfig(types.Starlark)
	cfg.SetHandler(logHandler)
	cfg.SetLoader(l)

	// Set the data provider
	cfg.SetDataProvider(buildCompositeDataProvider(staticData))

	// Create an evaluator using our custom createStarlarkEvaluator function
	return createStarlarkEvaluator(
		cfg,
		[]starlark.CompilerOption{starlark.WithGlobals([]string{constants.Ctx})},
	)
}

// fromFileLoader creates an evaluator from a file path using the specified machine type
func fromFileLoader(
	machineType types.Type,
	filePath string,
	opts ...any,
) (engine.EvaluatorWithPrep, error) {
	// Create a file loader
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	// Combine options, adding the loader
	allOpts := append([]any{options.WithLoader(l)}, opts...)

	return NewEvaluator(machineType, allOpts...)
}

// fromStringLoader creates an evaluator from a string content using the specified machine type
func fromStringLoader(
	machineType types.Type,
	content string,
	opts ...any,
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
	allOpts := append([]any{options.WithLoader(l)}, opts...)

	return NewEvaluator(machineType, allOpts...)
}

// NewEvaluator creates a new evaluator for the specified machine type.
//
// This function initializes a configuration with machine-specific defaults,
// applies the provided options to customize the configuration, validates the
// resulting configuration, and then creates an evaluator using the finalized
// configuration.
//
// Parameters:
//   - machineType: The type of machine (e.g., Extism, Risor, Starlark) for which
//     the evaluator is being created.
//   - opts: A variadic list of options to customize the evaluator's configuration.
//     These can be either engine.Option or machine-specific options.
//
// Returns:
//   - engine.EvaluatorWithPrep: The created evaluator, which includes preparation
//     capabilities for runtime data.
//   - error: An error if the configuration is invalid or if the evaluator creation
//     fails.
//
// Example usage:
//
//	evaluator, err := NewEvaluator(types.Risor, options.WithLoader(loader), risor.WithCtxGlobal())
//	if err != nil {
//	    log.Fatalf("Failed to create evaluator: %v", err)
//	}
func NewEvaluator(
	machineType types.Type,
	opts ...any,
) (engine.EvaluatorWithPrep, error) {
	switch machineType {
	case types.Extism:
		return NewExtismEvaluator(opts...)
	case types.Risor:
		return NewRisorEvaluator(opts...)
	case types.Starlark:
		return NewStarlarkEvaluator(opts...)
	default:
		return nil, fmt.Errorf("unsupported machine type: %s", machineType)
	}
}

// NewExtismEvaluator creates a new evaluator for Extism WASM
func NewExtismEvaluator(opts ...any) (engine.EvaluatorWithPrep, error) {
	// First create a config with the engine options
	cfg := options.DefaultConfig(types.Extism)

	// Separate engine options from machine options
	var engineOpts []options.Option
	var machineOpts []extism.CompilerOption

	for _, opt := range opts {
		switch o := opt.(type) {
		case options.Option:
			engineOpts = append(engineOpts, o)
		case extism.CompilerOption:
			machineOpts = append(machineOpts, o)
		default:
			return nil, fmt.Errorf("unsupported option type: %T", opt)
		}
	}

	// Apply all engine options
	for _, opt := range engineOpts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("error applying option: %w", err)
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Create the evaluator using our helper
	return createExtismEvaluator(cfg, machineOpts)
}

// createExtismEvaluator creates an Extism evaluator with the given options
func createExtismEvaluator(
	cfg *options.Config,
	compilerOptions []extism.CompilerOption,
) (engine.EvaluatorWithPrep, error) {
	// Create compiler using machine-specific factory function
	compiler, err := machines.NewExtismCompiler(compilerOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Extism compiler: %w", err)
	}

	// Create executable unit ID from source URL
	execUnitID := ""
	sourceURL := cfg.GetLoader().GetSourceURL()
	if sourceURL != nil {
		execUnitID = sourceURL.String()
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

// NewRisorEvaluator creates a new evaluator for Risor scripts
func NewRisorEvaluator(opts ...any) (engine.EvaluatorWithPrep, error) {
	// First create a config with the engine options
	cfg := options.DefaultConfig(types.Risor)

	// Separate engine options from machine options
	var engineOpts []options.Option
	var machineOpts []risor.CompilerOption

	for _, opt := range opts {
		switch o := opt.(type) {
		case options.Option:
			engineOpts = append(engineOpts, o)
		case risor.CompilerOption:
			machineOpts = append(machineOpts, o)
		default:
			return nil, fmt.Errorf("unsupported option type: %T", opt)
		}
	}

	// Apply all engine options
	for _, opt := range engineOpts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("error applying option: %w", err)
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Create the evaluator using our helper
	return createRisorEvaluator(cfg, machineOpts)
}

// createRisorEvaluator creates a Risor evaluator with the given options
func createRisorEvaluator(
	cfg *options.Config,
	compilerOptions []risor.CompilerOption,
) (engine.EvaluatorWithPrep, error) {
	// Create compiler using machine-specific factory function
	compiler, err := machines.NewRisorCompiler(compilerOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Risor compiler: %w", err)
	}

	// Create executable unit ID from source URL
	execUnitID := ""
	sourceURL := cfg.GetLoader().GetSourceURL()
	if sourceURL != nil {
		execUnitID = sourceURL.String()
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

// NewStarlarkEvaluator creates a new evaluator for Starlark scripts
func NewStarlarkEvaluator(opts ...any) (engine.EvaluatorWithPrep, error) {
	// First create a config with the engine options
	cfg := options.DefaultConfig(types.Starlark)

	// Separate engine options from machine options
	var engineOpts []options.Option
	var machineOpts []starlark.CompilerOption

	for _, opt := range opts {
		switch o := opt.(type) {
		case options.Option:
			engineOpts = append(engineOpts, o)
		case starlark.CompilerOption:
			machineOpts = append(machineOpts, o)
		default:
			return nil, fmt.Errorf("unsupported option type: %T", opt)
		}
	}

	// Apply all engine options
	for _, opt := range engineOpts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("error applying option: %w", err)
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Create the evaluator using our helper
	return createStarlarkEvaluator(cfg, machineOpts)
}

// createStarlarkEvaluator creates a Starlark evaluator with the given options
func createStarlarkEvaluator(
	cfg *options.Config,
	compilerOptions []starlark.CompilerOption,
) (engine.EvaluatorWithPrep, error) {
	// Create compiler using machine-specific factory function
	compiler, err := machines.NewStarlarkCompiler(compilerOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Starlark compiler: %w", err)
	}

	// Create executable unit ID from source URL
	execUnitID := ""
	sourceURL := cfg.GetLoader().GetSourceURL()
	if sourceURL != nil {
		execUnitID = sourceURL.String()
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

// buildCompositeDataProvider creates a composite data provider from a static data map
func buildCompositeDataProvider(
	staticData map[string]any,
) data.Provider {
	staticProvider := data.NewStaticProvider(staticData)
	dynamicProvider := data.NewContextProvider(constants.EvalData)
	return data.NewCompositeProvider(staticProvider, dynamicProvider)
}
