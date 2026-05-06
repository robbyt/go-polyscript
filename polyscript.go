// Package polyscript provides a unified interface for executing scripts in different language runtimes.
//
// Supported engines:
//   - Risor: Risor scripting language (https://risor.io)
//   - Starlark: Starlark configuration language (https://github.com/google/starlark-go)
//   - Extism: WebAssembly modules via the Extism runtime (https://extism.org)
//
// # Quick start
//
// The recommended API is the engine constructor + options pattern:
//
//	eval, err := polyscript.Risor(polyscript.FromString(script))
//	if err != nil {
//	    return err
//	}
//	result, err := eval.Eval(ctx)
//
// Static data attached at construction:
//
//	eval, err := polyscript.Risor(
//	    polyscript.FromString(script),
//	    polyscript.WithStaticData(map[string]any{"name": "World"}),
//	)
//
// WASM with Extism (entry point is required):
//
//	eval, err := polyscript.Extism(
//	    polyscript.FromBytes(wasmBytes),
//	    polyscript.WithEntryPoint("greet"),
//	)
//
// All engine constructors return a [platform.Evaluator]. For direct access to
// the underlying engine, see the per-engine packages under engines/.
package polyscript

import (
	"errors"
	"fmt"
	"log/slog"

	extismMachine "github.com/robbyt/go-polyscript/engines/extism"
	risorMachine "github.com/robbyt/go-polyscript/engines/risor"
	starlarkMachine "github.com/robbyt/go-polyscript/engines/starlark"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

// Source is an opaque script source produced by [FromString], [FromBytes],
// [FromFile], or [FromLoader]. Construction errors are deferred until the
// engine constructor (e.g. [Risor]) is called, so source helpers can be used
// inline.
type Source struct {
	build func() (loader.Loader, error)
}

// FromString returns a [Source] backed by an in-memory string. Most useful
// for Risor and Starlark scripts.
func FromString(content string) Source {
	return Source{build: func() (loader.Loader, error) {
		return loader.NewFromString(content)
	}}
}

// FromBytes returns a [Source] backed by an in-memory byte slice. Typically
// used for WASM modules consumed by [Extism].
func FromBytes(b []byte) Source {
	return Source{build: func() (loader.Loader, error) {
		return loader.NewFromBytes(b)
	}}
}

// FromFile returns a [Source] that reads from an absolute filesystem path.
func FromFile(path string) Source {
	return Source{build: func() (loader.Loader, error) {
		return loader.NewFromDisk(path)
	}}
}

// FromLoader wraps a custom [loader.Loader] (e.g. an HTTP loader created via
// [loader.NewFromHTTP]) so it can be used with the engine constructors.
func FromLoader(l loader.Loader) Source {
	return Source{build: func() (loader.Loader, error) {
		if l == nil {
			return nil, errors.New("polyscript: nil loader")
		}
		return l, nil
	}}
}

// Option configures evaluator construction.
type Option func(*config)

type config struct {
	handler    slog.Handler
	staticData map[string]any
	entryPoint string
}

// WithStaticData attaches a fixed map of values that the script will see
// alongside any runtime data added later via Evaluator.AddDataToContext.
func WithStaticData(data map[string]any) Option {
	return func(c *config) { c.staticData = data }
}

// WithLogHandler sets the slog.Handler used for diagnostic logging by the
// evaluator. If unset, the underlying engine picks a default.
func WithLogHandler(h slog.Handler) Option {
	return func(c *config) { c.handler = h }
}

// WithEntryPoint sets the WASM function name to invoke. Required for [Extism];
// ignored by other engines.
func WithEntryPoint(name string) Option {
	return func(c *config) { c.entryPoint = name }
}

func newConfig(opts []Option) *config {
	c := &config{}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	return c
}

// resolve runs the deferred Source builder.
func (s Source) resolve() (loader.Loader, error) {
	if s.build == nil {
		return nil, errors.New("polyscript: zero-value Source; use FromString/FromBytes/FromFile/FromLoader")
	}
	return s.build()
}

// Risor creates a Risor evaluator from the given source.
func Risor(src Source, opts ...Option) (platform.Evaluator, error) {
	ldr, err := src.resolve()
	if err != nil {
		return nil, err
	}
	cfg := newConfig(opts)
	if cfg.staticData != nil {
		return risorMachine.FromRisorLoaderWithData(cfg.handler, ldr, cfg.staticData)
	}
	return risorMachine.FromRisorLoader(cfg.handler, ldr)
}

// Starlark creates a Starlark evaluator from the given source.
func Starlark(src Source, opts ...Option) (platform.Evaluator, error) {
	ldr, err := src.resolve()
	if err != nil {
		return nil, err
	}
	cfg := newConfig(opts)
	if cfg.staticData != nil {
		return starlarkMachine.FromStarlarkLoaderWithData(cfg.handler, ldr, cfg.staticData)
	}
	return starlarkMachine.FromStarlarkLoader(cfg.handler, ldr)
}

// Extism creates an Extism (WASM) evaluator from the given source.
// The entry point is required and must be provided via [WithEntryPoint].
func Extism(src Source, opts ...Option) (platform.Evaluator, error) {
	cfg := newConfig(opts)
	if cfg.entryPoint == "" {
		return nil, fmt.Errorf("polyscript.Extism: entry point is required (use WithEntryPoint)")
	}
	ldr, err := src.resolve()
	if err != nil {
		return nil, err
	}
	if cfg.staticData != nil {
		return extismMachine.FromExtismLoaderWithData(cfg.handler, ldr, cfg.staticData, cfg.entryPoint)
	}
	return extismMachine.FromExtismLoader(cfg.handler, ldr, cfg.entryPoint)
}

// ----------------------------------------------------------------------------
// Deprecated convenience constructors.
//
// The functions below predate the [Risor]/[Starlark]/[Extism] + Source/Option
// pattern. They are retained for backwards compatibility and slated for
// removal in v1. New code should prefer the engine + Option style above.
// ----------------------------------------------------------------------------

// FromExtismFile creates an Extism evaluator from a WASM file.
//
// Deprecated: use [Extism] with [FromFile] and [WithEntryPoint] instead:
//
//	eval, err := polyscript.Extism(
//	    polyscript.FromFile(filePath),
//	    polyscript.WithLogHandler(logHandler),
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

	return extismMachine.FromExtismLoader(logHandler, l, entryPoint)
}

// FromExtismFileWithData creates an Extism evaluator with both static and
// dynamic data capabilities.
//
// Deprecated: use [Extism] with [FromFile], [WithStaticData] and
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

	return extismMachine.FromExtismLoaderWithData(logHandler, l, staticData, entryPoint)
}

// FromExtismBytes creates an Extism evaluator from WASM bytecode.
//
// Deprecated: use [Extism] with [FromBytes] and [WithEntryPoint] instead.
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

// FromExtismBytesWithData creates an Extism evaluator from WASM bytecode with
// both static and dynamic data capabilities.
//
// Deprecated: use [Extism] with [FromBytes], [WithStaticData] and
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

	return extismMachine.FromExtismLoaderWithData(logHandler, l, staticData, entryPoint)
}

// FromRisorFile creates a Risor evaluator from a .risor file.
//
// Deprecated: use [Risor] with [FromFile] instead.
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

// FromRisorFileWithData creates a Risor evaluator with both static and dynamic
// data capabilities.
//
// Deprecated: use [Risor] with [FromFile] and [WithStaticData] instead.
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
// Deprecated: use [Risor] with [FromString] instead.
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

// FromRisorStringWithData creates a Risor evaluator with both static and
// dynamic data capabilities.
//
// Deprecated: use [Risor] with [FromString] and [WithStaticData] instead.
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
// Deprecated: use [Starlark] with [FromFile] instead.
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

// FromStarlarkFileWithData creates a Starlark evaluator with both static and
// dynamic data capabilities.
//
// Deprecated: use [Starlark] with [FromFile] and [WithStaticData] instead.
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
// Deprecated: use [Starlark] with [FromString] instead.
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

// FromStarlarkStringWithData creates a Starlark evaluator with both static and
// dynamic data capabilities.
//
// Deprecated: use [Starlark] with [FromString] and [WithStaticData] instead.
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
