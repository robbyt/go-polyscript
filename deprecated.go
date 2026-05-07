// Deprecated convenience constructors.
//
// The functions in this file predate [polyscript.New] and the [polyscript.Option] type.
// They are retained for backwards compatibility and slated for removal in v1.
// New code should prefer the generic [polyscript.New] constructor — see polyscript.go
// for the supported pattern.
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
// Deprecated: use [New] with [Extism], [FromFile], and [WithEntryPoint] instead:
//
//	eval, err := polyscript.New[polyscript.Extism](
//	    polyscript.FromFile(filePath),
//	    polyscript.WithLogHandler[polyscript.Extism](logHandler),
//	    polyscript.WithEntryPoint(entryPoint),
//	)
func FromExtismFile(
	filePath string,
	logHandler slog.Handler,
	entryPoint string,
) (platform.Evaluator, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	return extismMachine.FromExtismLoader(l,
		extismMachine.WithLogHandler(logHandler),
		extismMachine.WithEntryPoint(entryPoint),
	)
}

// FromExtismFileWithData creates an Extism evaluator with both static and
// dynamic data capabilities.
//
// Deprecated: use [New] with [Extism], [FromFile], [WithStaticData], and
// [WithEntryPoint] instead.
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

	return extismMachine.FromExtismLoader(l,
		extismMachine.WithLogHandler(logHandler),
		extismMachine.WithStaticData(staticData),
		extismMachine.WithEntryPoint(entryPoint),
	)
}

// FromExtismBytes creates an Extism evaluator from WASM bytecode.
//
// Deprecated: use [New] with [Extism], [FromBytes], and [WithEntryPoint] instead.
func FromExtismBytes(
	wasmBytes []byte,
	logHandler slog.Handler,
	entryPoint string,
) (platform.Evaluator, error) {
	l, err := loader.NewFromBytes(wasmBytes)
	if err != nil {
		return nil, err
	}

	return extismMachine.FromExtismLoader(l,
		extismMachine.WithLogHandler(logHandler),
		extismMachine.WithEntryPoint(entryPoint),
	)
}

// FromExtismBytesWithData creates an Extism evaluator from WASM bytecode with
// both static and dynamic data capabilities.
//
// Deprecated: use [New] with [Extism], [FromBytes], [WithStaticData], and
// [WithEntryPoint] instead.
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

	return extismMachine.FromExtismLoader(l,
		extismMachine.WithLogHandler(logHandler),
		extismMachine.WithStaticData(staticData),
		extismMachine.WithEntryPoint(entryPoint),
	)
}

// FromRisorFile creates a Risor evaluator from a .risor file.
//
// Deprecated: use [New] with [Risor] and [FromFile] instead.
func FromRisorFile(
	filePath string,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	return risorMachine.FromRisorLoader(l, risorMachine.WithLogHandler(logHandler))
}

// FromRisorFileWithData creates a Risor evaluator with both static and dynamic
// data capabilities.
//
// Deprecated: use [New] with [Risor], [FromFile], and [WithStaticData] instead.
func FromRisorFileWithData(
	filePath string,
	staticData map[string]any,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	return risorMachine.FromRisorLoader(l,
		risorMachine.WithLogHandler(logHandler),
		risorMachine.WithStaticData(staticData),
	)
}

// FromRisorString creates a Risor evaluator from a script string.
//
// Deprecated: use [New] with [Risor] and [FromString] instead.
func FromRisorString(
	content string,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromString(content)
	if err != nil {
		return nil, err
	}

	return risorMachine.FromRisorLoader(l, risorMachine.WithLogHandler(logHandler))
}

// FromRisorStringWithData creates a Risor evaluator with both static and
// dynamic data capabilities.
//
// Deprecated: use [New] with [Risor], [FromString], and [WithStaticData] instead.
func FromRisorStringWithData(
	script string,
	staticData map[string]any,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromString(script)
	if err != nil {
		return nil, err
	}

	return risorMachine.FromRisorLoader(l,
		risorMachine.WithLogHandler(logHandler),
		risorMachine.WithStaticData(staticData),
	)
}

// FromStarlarkFile creates a Starlark evaluator from a .star file.
//
// Deprecated: use [New] with [Starlark] and [FromFile] instead.
func FromStarlarkFile(
	filePath string,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	return starlarkMachine.FromStarlarkLoader(l, starlarkMachine.WithLogHandler(logHandler))
}

// FromStarlarkFileWithData creates a Starlark evaluator with both static and
// dynamic data capabilities.
//
// Deprecated: use [New] with [Starlark], [FromFile], and [WithStaticData] instead.
func FromStarlarkFileWithData(
	filePath string,
	staticData map[string]any,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	return starlarkMachine.FromStarlarkLoader(l,
		starlarkMachine.WithLogHandler(logHandler),
		starlarkMachine.WithStaticData(staticData),
	)
}

// FromStarlarkString creates a Starlark evaluator from a script string.
//
// Deprecated: use [New] with [Starlark] and [FromString] instead.
func FromStarlarkString(
	content string,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromString(content)
	if err != nil {
		return nil, err
	}

	return starlarkMachine.FromStarlarkLoader(l, starlarkMachine.WithLogHandler(logHandler))
}

// FromStarlarkStringWithData creates a Starlark evaluator with both static and
// dynamic data capabilities.
//
// Deprecated: use [New] with [Starlark], [FromString], and [WithStaticData] instead.
func FromStarlarkStringWithData(
	script string,
	staticData map[string]any,
	logHandler slog.Handler,
) (platform.Evaluator, error) {
	l, err := loader.NewFromString(script)
	if err != nil {
		return nil, err
	}

	return starlarkMachine.FromStarlarkLoader(l,
		starlarkMachine.WithLogHandler(logHandler),
		starlarkMachine.WithStaticData(staticData),
	)
}
