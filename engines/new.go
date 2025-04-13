//go:generate go run ./types/gen/typeGen.go
// Code generated by go generate; DO NOT EDIT.

package engines

import (
	"fmt"
	"log/slog"

	extismCompiler "github.com/robbyt/go-polyscript/engines/extism/compiler"
	extismEvaluator "github.com/robbyt/go-polyscript/engines/extism/evaluator"
	risorCompiler "github.com/robbyt/go-polyscript/engines/risor/compiler"
	risorEvaluator "github.com/robbyt/go-polyscript/engines/risor/evaluator"
	starlarkCompiler "github.com/robbyt/go-polyscript/engines/starlark/compiler"
	starlarkEvaluator "github.com/robbyt/go-polyscript/engines/starlark/evaluator"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/script"

	machineTypes "github.com/robbyt/go-polyscript/engines/types"
)

// NewEvaluator creates a new VM with the given CPU type and globals.
// This will load a script from a ExecutableUnit object into the VM, and can be run immediately.
// The ExecutableUnit contains a DataProvider that provides runtime data for evaluation.
func NewEvaluator(handler slog.Handler, ver *script.ExecutableUnit) (platform.Evaluator, error) {
	if ver == nil {
		return nil, fmt.Errorf("version is nil")
	}

	switch ver.GetMachineType() {
	case machineTypes.Risor:
		// Risor VM: https://github.com/risor-io/risor
		return risorEvaluator.New(handler, ver), nil
	case machineTypes.Starlark:
		// Starlark VM: https://github.com/google/starlark-go
		return starlarkEvaluator.New(handler, ver), nil
	case machineTypes.Extism:
		// Extism WASM VM: https://extism.org/
		return extismEvaluator.New(handler, ver), nil
	default:
		return nil, fmt.Errorf("%w: %s", machineTypes.ErrInvalidMachineType, ver.GetMachineType())
	}
}

// NewCompiler creates a compiler based on the option types.
// It determines which compiler to use by checking the types of the provided options.
func NewCompiler(opts ...any) (script.Compiler, error) {
	if len(opts) == 0 {
		return nil, fmt.Errorf("no options provided")
	}

	// Check for Risor options
	{
		var risorOpts []risorCompiler.FunctionalOption
		allMatch := true

		for _, opt := range opts {
			if o, ok := opt.(risorCompiler.FunctionalOption); ok {
				risorOpts = append(risorOpts, o)
			} else {
				allMatch = false
				break
			}
		}

		if allMatch && len(risorOpts) > 0 {
			compiler, err := risorCompiler.New(risorOpts...)
			if err != nil {
				return nil, fmt.Errorf("failed to create Risor compiler: %w", err)
			}
			return compiler, nil
		}
	}

	// Check for Starlark options
	{
		var starlarkOpts []starlarkCompiler.FunctionalOption
		allMatch := true

		for _, opt := range opts {
			if o, ok := opt.(starlarkCompiler.FunctionalOption); ok {
				starlarkOpts = append(starlarkOpts, o)
			} else {
				allMatch = false
				break
			}
		}

		if allMatch && len(starlarkOpts) > 0 {
			compiler, err := starlarkCompiler.New(starlarkOpts...)
			if err != nil {
				return nil, fmt.Errorf("failed to create Starlark compiler: %w", err)
			}
			return compiler, nil
		}
	}

	// Check for Extism options
	{
		var extismOpts []extismCompiler.FunctionalOption
		allMatch := true

		for _, opt := range opts {
			if o, ok := opt.(extismCompiler.FunctionalOption); ok {
				extismOpts = append(extismOpts, o)
			} else {
				allMatch = false
				break
			}
		}

		if allMatch && len(extismOpts) > 0 {
			compiler, err := extismCompiler.New(extismOpts...)
			if err != nil {
				return nil, fmt.Errorf("failed to create Extism compiler: %w", err)
			}
			return compiler, nil
		}
	}

	return nil, fmt.Errorf("unable to determine compiler type from provided options")
}
