//go:generate go run ./types/gen/typeGen.go
// Code generated by go generate; DO NOT EDIT.

package machines

import (
	"fmt"
	"log/slog"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/script"
	extismMachine "github.com/robbyt/go-polyscript/machines/extism"
	risorMachine "github.com/robbyt/go-polyscript/machines/risor"
	starlarkMachine "github.com/robbyt/go-polyscript/machines/starlark"

	machineTypes "github.com/robbyt/go-polyscript/machines/types"
)

// NewEvaluator creates a new VM with the given CPU type and globals
// This will load a script from a ExecutableUnit object into the VM, and can be run immediately.
// The ExecutableUnit contains a DataProvider that provides runtime data for evaluation.
func NewEvaluator(handler slog.Handler, ver *script.ExecutableUnit) (engine.Evaluator, error) {
	if ver == nil {
		return nil, fmt.Errorf("version is nil")
	}

	switch ver.GetMachineType() {
	case machineTypes.Risor:
		// Risor VM: https://github.com/risor-io/risor
		return risorMachine.NewBytecodeEvaluator(handler, ver), nil
	case machineTypes.Starlark:
		// Starlark VM: https://github.com/google/starlark-go
		return starlarkMachine.NewBytecodeEvaluator(handler, ver), nil
	case machineTypes.Extism:
		// Extism WASM VM: https://extism.org/
		return extismMachine.NewBytecodeEvaluator(handler, ver), nil
	default:
		return nil, fmt.Errorf("%w: %s", machineTypes.ErrInvalidMachineType, ver.GetMachineType())
	}
}

// NewCompiler creates a compiler for the specified machine type with given globals
func NewCompiler(handler slog.Handler, machineType machineTypes.Type, compilerOptions any) (script.Compiler, error) {
	switch machineType {
	case machineTypes.Risor:
		// Risor VM: https://github.com/risor-io/risor
		risorOptions, ok := compilerOptions.(risorMachine.CompilerOptions)
		if !ok {
			return nil, fmt.Errorf("invalid compiler options for Risor machine, got %T", compilerOptions)
		}
		return risorMachine.NewCompiler(handler, risorOptions), nil
	case machineTypes.Starlark:
		// Starlark VM: https://github.com/google/starlark-go
		starlarkOptions, ok := compilerOptions.(starlarkMachine.CompilerOptions)
		if !ok {
			return nil, fmt.Errorf("invalid compiler options for Starlark machine, got %T", compilerOptions)
		}
		return starlarkMachine.NewCompiler(handler, starlarkOptions), nil
	case machineTypes.Extism:
		// Extism WASM VM: https://extism.org/
		extismOptions, ok := compilerOptions.(extismMachine.CompilerOptions)
		if !ok {
			return nil, fmt.Errorf("invalid compiler options for Extism machine, got %T", compilerOptions)
		}
		return extismMachine.NewCompiler(handler, extismOptions), nil
	default:
		return nil, fmt.Errorf("unsupported machine type: %s", machineType)
	}
}
