package polyscript

import (
	"log/slog"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	extismMachine "github.com/robbyt/go-polyscript/machines/extism"
	risorMachine "github.com/robbyt/go-polyscript/machines/risor"
	starlarkMachine "github.com/robbyt/go-polyscript/machines/starlark"
)

// FromExtismFile creates an Extism evaluator from a WASM file
func FromExtismFile(
	filePath string,
	logHandler slog.Handler,
	entryPoint string,
) (engine.EvaluatorWithPrep, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	return extismMachine.FromExtismLoader(logHandler, l, entryPoint)
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

	return extismMachine.FromExtismLoaderWithData(logHandler, l, staticData, entryPoint)
}

// FromRisorFile creates a Risor evaluator from a .risor file
func FromRisorFile(filePath string, logHandler slog.Handler) (engine.EvaluatorWithPrep, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	return risorMachine.FromRisorLoader(logHandler, l)
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

	return risorMachine.FromRisorLoaderWithData(logHandler, l, staticData)
}

// FromRisorString creates a Risor evaluator from a script string
func FromRisorString(content string, logHandler slog.Handler) (engine.EvaluatorWithPrep, error) {
	l, err := loader.NewFromString(content)
	if err != nil {
		return nil, err
	}

	return risorMachine.FromRisorLoader(logHandler, l)
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

	return risorMachine.FromRisorLoaderWithData(logHandler, l, staticData)
}

// FromStarlarkFile creates a Starlark evaluator from a .star file
func FromStarlarkFile(filePath string, logHandler slog.Handler) (engine.EvaluatorWithPrep, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	return starlarkMachine.FromStarlarkLoader(logHandler, l)
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

	return starlarkMachine.FromStarlarkLoaderWithData(logHandler, l, staticData)
}

// FromStarlarkString creates a Starlark evaluator from a script string
func FromStarlarkString(content string, logHandler slog.Handler) (engine.EvaluatorWithPrep, error) {
	l, err := loader.NewFromString(content)
	if err != nil {
		return nil, err
	}

	return starlarkMachine.FromStarlarkLoader(logHandler, l)
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

	return starlarkMachine.FromStarlarkLoaderWithData(logHandler, l, staticData)
}
