// Package polyscript provides a unified interface for executing scripts in different language runtimes.
//
// This package supports these "engine" types:
//   - Extism: WebAssembly modules
//   - Risor: Risor scripting language
//   - Starlark: Starlark configuration language
//
// For each engine, there are two main patterns available:
//  1. Basic execution: Load and execute scripts without external data
//  2. With data preparation: Provide initial static data, and thread-safe dynamic runtime data
//
// All functions in this package return a common evaluation.Evaluator interface. For direct
// access to the underlying engine, use the specific engine's methods.
package polyscript

import (
	"log/slog"

	extismMachine "github.com/robbyt/go-polyscript/engines/extism"
	risorMachine "github.com/robbyt/go-polyscript/engines/risor"
	starlarkMachine "github.com/robbyt/go-polyscript/engines/starlark"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

// FromExtismFile creates an Extism evaluator from a WASM file.
//
// Example:
//
//	be, err := FromExtismFile("path/to/module.wasm", slog.Default().Handler(), "process")
//	result, err := be.Eval(context.Background())
func FromExtismFile(
	filePath string,
	logHandler slog.Handler,
	entryPoint string,
) (platform.Evaluator, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	return extismMachine.FromExtismLoader(logHandler, l, entryPoint)
}

// FromExtismFileWithData creates an Extism evaluator with both static and dynamic data capabilities.
// To add runtime data, use the AddDataToContext method on the evaluator to add data to the context.
//
// Example:
//
//	staticData := map[string]any{"config": "value"}
//	be, err := FromExtismFileWithData("path/to/module.wasm", staticData, slog.Default().Handler(), "process")
//
//	runtimeData := map[string]any{"request": req}
//	ctx, err = be.AddDataToContext(context.Background(), runtimeData)
//	result, err := be.Eval(ctx)
func FromExtismFileWithData(
	filePath string,
	staticData map[string]any,
	logHandler slog.Handler,
	entryPoint string,
) (platform.Evaluator, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	return extismMachine.FromExtismLoaderWithData(logHandler, l, staticData, entryPoint)
}

// FromExtismBytes creates an Extism evaluator from WASM bytecode.
//
// Example:
//
//	wasmBytes := []byte{...} // Your WASM bytecode
//	be, err := FromExtismBytes(wasmBytes, slog.Default().Handler(), "process")
func FromExtismBytes(
	wasmBytes []byte,
	logHandler slog.Handler,
	entryPoint string,
) (platform.Evaluator, error) {
	l, err := loader.NewFromBytes(wasmBytes)
	if err != nil {
		return nil, err
	}

	return extismMachine.FromExtismLoader(logHandler, l, entryPoint)
}

// FromExtismBytesWithData creates an Extism evaluator from WASM bytecode with both static and dynamic data capabilities.
// To add runtime data, use the AddDataToContext method on the evaluator to add data to the context.
//
// Example:
//
//	wasmBytes := []byte{...} // Your WASM bytecode
//	staticData := map[string]any{"config": "value"}
//	be, err := FromExtismBytesWithData(wasmBytes, staticData, slog.Default().Handler(), "process")
//
//	runtimeData := map[string]any{"request": req}
//	ctx, err = be.AddDataToContext(context.Background(), runtimeData)
//	result, err := be.Eval(ctx)
func FromExtismBytesWithData(
	wasmBytes []byte,
	staticData map[string]any,
	logHandler slog.Handler,
	entryPoint string,
) (platform.Evaluator, error) {
	l, err := loader.NewFromBytes(wasmBytes)
	if err != nil {
		return nil, err
	}

	return extismMachine.FromExtismLoaderWithData(logHandler, l, staticData, entryPoint)
}

// FromRisorFile creates a Risor evaluator from a .risor file.
//
// Example:
//
//	be, err := FromRisorFile("path/to/script.risor", slog.Default().Handler())
//	result, err := be.Eval(context.Background())
func FromRisorFile(
	filePath string,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	return risorMachine.FromRisorLoader(logHandler, l)
}

// FromRisorFileWithData creates a Risor evaluator with both static and dynamic data capabilities.
// To add runtime data, use the AddDataToContext method on the evaluator to add data to the context.
//
// Example:
//
//	staticData := map[string]any{"config": "value"}
//	be, err := FromRisorFileWithData("path/to/script.risor", staticData, slog.Default().Handler())
//
//	runtimeData := map[string]any{"request": req}
//	ctx, err = be.AddDataToContext(context.Background(), runtimeData)
//	result, err := be.Eval(ctx)
func FromRisorFileWithData(
	filePath string,
	staticData map[string]any,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	return risorMachine.FromRisorLoaderWithData(logHandler, l, staticData)
}

// FromRisorString creates a Risor evaluator from a script string.
//
// Example:
//
//	script := `return "Hello, world!"`
//	be, err := FromRisorString(script, slog.Default().Handler())
//	result, err := be.Eval(context.Background())
func FromRisorString(
	content string,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromString(content)
	if err != nil {
		return nil, err
	}

	return risorMachine.FromRisorLoader(logHandler, l)
}

// FromRisorStringWithData creates a Risor evaluator with both static and dynamic data capabilities.
// To add runtime data, use the AddDataToContext method on the evaluator to add data to the context.
//
// Example:
//
//	script := `return config + " and " + request.field`
//	staticData := map[string]any{"config": "static value"}
//	be, err := FromRisorStringWithData(script, staticData, slog.Default().Handler())
//
//	runtimeData := map[string]any{"request": map[string]string{"field": "dynamic value"}}
//	ctx, err = be.AddDataToContext(context.Background(), runtimeData)
//	result, err := be.Eval(ctx)
func FromRisorStringWithData(
	script string,
	staticData map[string]any,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromString(script)
	if err != nil {
		return nil, err
	}

	return risorMachine.FromRisorLoaderWithData(logHandler, l, staticData)
}

// FromStarlarkFile creates a Starlark evaluator from a .star file.
//
// Example:
//
//	be, err := FromStarlarkFile("path/to/script.star", slog.Default().Handler())
//	result, err := be.Eval(context.Background())
func FromStarlarkFile(
	filePath string,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	return starlarkMachine.FromStarlarkLoader(logHandler, l)
}

// FromStarlarkFileWithData creates a Starlark evaluator with both static and dynamic data capabilities.
// To add runtime data, use the AddDataToContext method on the evaluator to add data to the context.
//
// Example:
//
//	staticData := map[string]any{"constants": map[string]string{"version": "1.0"}}
//	be, err := FromStarlarkFileWithData("path/to/script.star", staticData, slog.Default().Handler())
//
//	runtimeData := map[string]any{"input": userInput}
//	ctx, err = be.AddDataToContext(context.Background(), runtimeData)
//	result, err := be.Eval(ctx)
func FromStarlarkFileWithData(
	filePath string,
	staticData map[string]any,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	return starlarkMachine.FromStarlarkLoaderWithData(logHandler, l, staticData)
}

// FromStarlarkString creates a Starlark evaluator from a script string.
//
// Example:
//
//	script := `def main(): return "Hello from Starlark"`
//	be, err := FromStarlarkString(script, slog.Default().Handler())
//	result, err := be.Eval(context.Background())
func FromStarlarkString(
	content string,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromString(content)
	if err != nil {
		return nil, err
	}

	return starlarkMachine.FromStarlarkLoader(logHandler, l)
}

// FromStarlarkStringWithData creates a Starlark evaluator with both static and dynamic data
// capabilities. To add runtime data, use the AddDataToContext method on the evaluator to add data
// to the context.
//
// Example:
//
//	script := `def main(): return constants.greeting + " " + user.name`
//	staticData := map[string]any{"constants": map[string]string{"greeting": "Hello"}}
//	be, err := FromStarlarkStringWithData(script, staticData, slog.Default().Handler())
//
//	runtimeData := map[string]any{"user": map[string]string{"name": "World"}}
//	ctx, err = be.AddDataToContext(context.Background(), runtimeData)
//	result, err := be.Eval(ctx)
func FromStarlarkStringWithData(
	script string,
	staticData map[string]any,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromString(script)
	if err != nil {
		return nil, err
	}

	return starlarkMachine.FromStarlarkLoaderWithData(logHandler, l, staticData)
}
