// Package polyscript provides a unified interface for executing scripts in different language runtimes.
//
// Supported engines:
//   - Risor: Risor scripting language (https://risor.io)
//   - Starlark: Starlark configuration language (https://github.com/google/starlark-go)
//   - Extism: WebAssembly modules via the Extism runtime (https://extism.org)
//
// # Quick start
//
// The recommended API is the generic [New] constructor with an engine type
// argument plus a [Source] and zero or more [Option]s:
//
//	eval, err := polyscript.New[polyscript.Risor](polyscript.FromString(script))
//	if err != nil {
//	    return err
//	}
//	result, err := eval.Eval(ctx)
//
// Static data attached at construction:
//
//	eval, err := polyscript.New[polyscript.Risor](
//	    polyscript.FromString(script),
//	    polyscript.WithStaticData[polyscript.Risor](map[string]any{"name": "World"}),
//	)
//
// WASM with Extism (entry point is required and bound to Extism at compile time):
//
//	eval, err := polyscript.New[polyscript.Extism](
//	    polyscript.FromBytes(wasmBytes),
//	    polyscript.WithEntryPoint("greet"),
//	)
//
// All engine constructors return a [platform.Evaluator]. For direct access to
// the underlying engine, see the per-engine packages under engines/.
//
// # A note on type arguments
//
// [WithStaticData] and [WithLogHandler] are generic over the engine type. Go's
// current type inference cannot propagate the [New] call's type argument into
// these helpers when the [Source] sits between them as a non-variadic
// parameter, so an explicit type argument is usually required:
//
//	polyscript.WithStaticData[polyscript.Risor](data)
//
// [WithEntryPoint] is bound to [Extism] alone and never needs a type argument.
// Passing it to [New[Risor]] or [New[Starlark]] is a compile-time error
// rather than a silent no-op.
package polyscript

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"

	extismMachine "github.com/robbyt/go-polyscript/engines/extism"
	risorMachine "github.com/robbyt/go-polyscript/engines/risor"
	starlarkMachine "github.com/robbyt/go-polyscript/engines/starlark"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

// ----------------------------------------------------------------------------
// Engine markers
// ----------------------------------------------------------------------------

// Engine is the sealed type set used as the type argument to [New]. The three
// concrete engines are [Risor], [Starlark], and [Extism]. The single
// unexported method is a marker that prevents external packages from adding
// their own engine types, so [Option] specialization stays exhaustive.
type Engine interface {
	polyscriptEngine()
}

// Risor selects the Risor scripting engine.
type Risor struct{}

// polyscriptEngine is a sealed marker; intentionally empty.
func (Risor) polyscriptEngine() {}

// Starlark selects the Starlark configuration language engine.
type Starlark struct{}

// polyscriptEngine is a sealed marker; intentionally empty.
func (Starlark) polyscriptEngine() {}

// Extism selects the Extism (WebAssembly) engine.
type Extism struct{}

// polyscriptEngine is a sealed marker; intentionally empty.
func (Extism) polyscriptEngine() {}

// ----------------------------------------------------------------------------
// Sources
// ----------------------------------------------------------------------------

// Source is an opaque script source produced by [FromString], [FromBytes],
// [FromFile], or [FromLoader]. Construction errors are deferred until the
// engine constructor (e.g. [New]) is called, so source helpers can be used
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
//
// The slice is copied so later mutations to the caller's buffer do not affect
// compilation.
func FromBytes(b []byte) Source {
	cp := make([]byte, len(b))
	copy(cp, b)
	return Source{build: func() (loader.Loader, error) {
		return loader.NewFromBytes(cp)
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
		if isNilLoader(l) {
			return nil, errors.New("polyscript: nil loader")
		}
		return l, nil
	}}
}

// isNilLoader reports whether l is nil, including the typed-nil case where
// the interface holds a non-nil type descriptor but a nil concrete value
// (e.g. var p *myLoader = nil; FromLoader(p)).
func isNilLoader(l loader.Loader) bool {
	if l == nil {
		return true
	}
	v := reflect.ValueOf(l)
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return v.IsNil()
	}
	return false
}

// resolve runs the deferred Source builder.
func (s Source) resolve() (loader.Loader, error) {
	if s.build == nil {
		return nil, errors.New("polyscript: zero-value Source; use FromString/FromBytes/FromFile/FromLoader")
	}
	return s.build()
}

// ----------------------------------------------------------------------------
// Options
// ----------------------------------------------------------------------------

// Option configures construction of an engine of type E. Engine-specific
// options (e.g. [WithEntryPoint]) bind to a single engine and are rejected
// at compile time when passed to [New] for the wrong engine. Shared options
// (e.g. [WithStaticData], [WithLogHandler]) are generic in E and work with
// any engine.
type Option[E Engine] func(*config)

type config struct {
	handler    slog.Handler
	staticData map[string]any
	entryPoint string
}

// WithStaticData attaches a fixed map of values that the script will see
// alongside any runtime data added later via Evaluator.AddDataToContext.
//
// The type parameter is normally inferred from the surrounding [New] call.
func WithStaticData[E Engine](data map[string]any) Option[E] {
	return func(c *config) { c.staticData = data }
}

// WithLogHandler sets the slog.Handler used for diagnostic logging by the
// evaluator. If unset, the underlying engine picks a default.
//
// The type parameter is normally inferred from the surrounding [New] call.
func WithLogHandler[E Engine](h slog.Handler) Option[E] {
	return func(c *config) { c.handler = h }
}

// WithEntryPoint sets the WASM function name to invoke. It is required for
// [Extism] and bound to it at compile time — passing it to [New[Risor]] or
// [New[Starlark]] is a compile error rather than a silent no-op.
func WithEntryPoint(name string) Option[Extism] {
	return func(c *config) { c.entryPoint = name }
}

// ----------------------------------------------------------------------------
// Constructor
// ----------------------------------------------------------------------------

// New constructs an evaluator for the engine selected via the type parameter:
//
//	eval, err := polyscript.New[polyscript.Risor](polyscript.FromString(script))
//
//	eval, err := polyscript.New[polyscript.Extism](
//	    polyscript.FromBytes(wasmBytes),
//	    polyscript.WithEntryPoint("greet"),
//	    polyscript.WithStaticData(map[string]any{"input": "World"}),
//	)
//
// Shared options ([WithStaticData], [WithLogHandler]) work for any engine.
// [WithEntryPoint] is bound to [Extism]; passing it to [Risor] or [Starlark]
// is a compile error.
func New[E Engine](src Source, opts ...Option[E]) (platform.Evaluator, error) {
	cfg := &config{}
	for _, o := range opts {
		if o != nil {
			o(cfg)
		}
	}
	var e E
	switch any(e).(type) {
	case Extism:
		if cfg.entryPoint == "" {
			return nil, errors.New("polyscript.New[Extism]: entry point is required (use WithEntryPoint)")
		}
	}

	ldr, err := src.resolve()
	if err != nil {
		return nil, err
	}

	switch any(e).(type) {
	case Risor:
		if cfg.staticData != nil {
			return risorMachine.FromRisorLoaderWithData(cfg.handler, ldr, cfg.staticData)
		}
		return risorMachine.FromRisorLoader(cfg.handler, ldr)
	case Starlark:
		if cfg.staticData != nil {
			return starlarkMachine.FromStarlarkLoaderWithData(cfg.handler, ldr, cfg.staticData)
		}
		return starlarkMachine.FromStarlarkLoader(cfg.handler, ldr)
	case Extism:
		if cfg.staticData != nil {
			return extismMachine.FromExtismLoaderWithData(cfg.handler, ldr, cfg.staticData, cfg.entryPoint)
		}
		return extismMachine.FromExtismLoader(cfg.handler, ldr, cfg.entryPoint)
	default:
		return nil, fmt.Errorf("polyscript.New: unsupported engine type %T", e)
	}
}

// ----------------------------------------------------------------------------
// Deprecated convenience constructors.
//
// The functions below predate [New] and the [Option] type. They are retained
// for backwards compatibility and slated for removal in v1. New code should
// prefer the generic [New] constructor.
// ----------------------------------------------------------------------------

// FromExtismFile creates an Extism evaluator from a WASM file.
//
// Deprecated: use [New] with [Extism], [FromFile], and [WithEntryPoint] instead:
//
//	eval, err := polyscript.New[polyscript.Extism](
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

	return extismMachine.FromExtismLoaderWithData(logHandler, l, staticData, entryPoint)
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

	return extismMachine.FromExtismLoader(logHandler, l, entryPoint)
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

	return extismMachine.FromExtismLoaderWithData(logHandler, l, staticData, entryPoint)
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

	return risorMachine.FromRisorLoader(logHandler, l)
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

	return risorMachine.FromRisorLoaderWithData(logHandler, l, staticData)
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

	return risorMachine.FromRisorLoader(logHandler, l)
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

	return risorMachine.FromRisorLoaderWithData(logHandler, l, staticData)
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

	return starlarkMachine.FromStarlarkLoader(logHandler, l)
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

	return starlarkMachine.FromStarlarkLoaderWithData(logHandler, l, staticData)
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

	return starlarkMachine.FromStarlarkLoader(logHandler, l)
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

	return starlarkMachine.FromStarlarkLoaderWithData(logHandler, l, staticData)
}
