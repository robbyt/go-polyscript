package polyscript

import (
	"context"
	"fmt"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/machines"
	"github.com/robbyt/go-polyscript/machines/types"
	"github.com/robbyt/go-polyscript/options"
)

// EvaluatorWrapper wraps a machine-specific evaluator and stores the ExecutableUnit.
// This allows callers to follow the "compile once, run many times" pattern.
// It implements both the Evaluator and EvalDataPreparer interfaces.
type EvaluatorWrapper struct {
	delegate engine.Evaluator
	execUnit *script.ExecutableUnit
}

// NewEvaluatorWrapper creates a new evaluator wrapper
func NewEvaluatorWrapper(
	delegate engine.Evaluator,
	execUnit *script.ExecutableUnit,
) engine.EvaluatorWithPrep {
	return &EvaluatorWrapper{
		delegate: delegate,
		execUnit: execUnit,
	}
}

// Eval implements the engine.Evaluator interface
// It delegates to the wrapped evaluator using the stored ExecutableUnit
func (e *EvaluatorWrapper) Eval(ctx context.Context) (engine.EvaluatorResponse, error) {
	return e.delegate.Eval(ctx)
}

// PrepareContext implements the engine.EvalDataPreparer interface by enriching
// the context with data for script evaluation. It delegates to the wrapped evaluator
// if it implements EvalDataPreparer, otherwise uses the ExecutableUnit's DataProvider directly.
func (e *EvaluatorWrapper) PrepareContext(
	ctx context.Context,
	data ...any,
) (context.Context, error) {
	// If the delegate implements EvalDataPreparer, use it
	if preparer, ok := e.delegate.(engine.EvalDataPreparer); ok {
		return preparer.PrepareContext(ctx, data...)
	}

	// Fallback implementation using the executable unit's data provider
	if e.execUnit == nil || e.execUnit.GetDataProvider() == nil {
		return ctx, fmt.Errorf("no data provider available")
	}

	return e.execUnit.GetDataProvider().AddDataToContext(ctx, data...)
}

// GetExecutableUnit returns the stored ExecutableUnit
// This is useful for examining or modifying the unit
func (e *EvaluatorWrapper) GetExecutableUnit() *script.ExecutableUnit {
	return e.execUnit
}

// WithExecutableUnit returns a new evaluator wrapper with the specified ExecutableUnit
// This is useful for creating evaluator variants with different data providers
func (e *EvaluatorWrapper) WithExecutableUnit(
	execUnit *script.ExecutableUnit,
) engine.EvaluatorWithPrep {
	return &EvaluatorWrapper{
		delegate: e.delegate,
		execUnit: execUnit,
	}
}

// NewStarlarkEvaluator creates a new evaluator for Starlark scripts
func NewStarlarkEvaluator(opts ...options.Option) (engine.EvaluatorWithPrep, error) {
	// Initialize with Starlark defaults
	cfg := options.DefaultConfig(types.Starlark)

	// Apply all options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("error applying option: %w", err)
		}
	}

	// Apply defaults option as final step to fill in any missing values
	if err := options.WithDefaults()(cfg); err != nil {
		return nil, fmt.Errorf("error applying defaults: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return createEvaluator(cfg)
}

// NewRisorEvaluator creates a new evaluator for Risor scripts
func NewRisorEvaluator(opts ...options.Option) (engine.EvaluatorWithPrep, error) {
	// Initialize with Risor defaults
	cfg := options.DefaultConfig(types.Risor)

	// Apply all options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("error applying option: %w", err)
		}
	}

	// Apply defaults option as final step to fill in any missing values
	if err := options.WithDefaults()(cfg); err != nil {
		return nil, fmt.Errorf("error applying defaults: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return createEvaluator(cfg)
}

// NewExtismEvaluator creates a new evaluator for Extism WASM
func NewExtismEvaluator(opts ...options.Option) (engine.EvaluatorWithPrep, error) {
	// Initialize with Extism defaults
	cfg := options.DefaultConfig(types.Extism)

	// Apply all options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("error applying option: %w", err)
		}
	}

	// Apply defaults option as final step to fill in any missing values
	if err := options.WithDefaults()(cfg); err != nil {
		return nil, fmt.Errorf("error applying defaults: %w", err)
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

// FromStarlarkString creates a Starlark evaluator from a script string
func FromStarlarkString(content string, opts ...options.Option) (engine.EvaluatorWithPrep, error) {
	// Create a string loader
	l, err := loader.NewFromString(content)
	if err != nil {
		return nil, err
	}

	// Combine options, adding the loader
	allOpts := append([]options.Option{options.WithLoader(l)}, opts...)

	return NewStarlarkEvaluator(allOpts...)
}

// FromRisorString creates a Risor evaluator from a script string
func FromRisorString(content string, opts ...options.Option) (engine.EvaluatorWithPrep, error) {
	// Create a string loader
	l, err := loader.NewFromString(content)
	if err != nil {
		return nil, err
	}

	// Combine options, adding the loader
	allOpts := append([]options.Option{options.WithLoader(l)}, opts...)

	return NewRisorEvaluator(allOpts...)
}

// FromExtismFile creates an Extism evaluator from a WASM file
func FromExtismFile(filePath string, opts ...options.Option) (engine.EvaluatorWithPrep, error) {
	// Create a file loader
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	// Combine options, adding the loader
	allOpts := append([]options.Option{options.WithLoader(l)}, opts...)

	return NewExtismEvaluator(allOpts...)
}
