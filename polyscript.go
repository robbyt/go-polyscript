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

// Engine is the type-set constraint used as the type argument to [New]. It is
// satisfied by exactly three concrete types: [Risor], [Starlark], and
// [Extism]. Using a type-set constraint means Engine is a compile-time
// gate — callers that try to instantiate [New] with any other type
// (including pointer types like *Risor) get a build error.
//
// Engine is a constraint, not a regular interface; values cannot be assigned
// to it.
type Engine interface {
	Risor | Starlark | Extism
}

// Risor selects the Risor scripting engine.
type Risor struct{}

// Starlark selects the Starlark configuration language engine.
type Starlark struct{}

// Extism selects the Extism (WebAssembly) engine.
type Extism struct{}

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
// evaluator. A nil handler is permitted and means "inherit from
// slog.Default()" — equivalent to omitting the option.
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
//	    polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "World"}),
//	)
//
// Shared options ([WithStaticData], [WithLogHandler]) work for any engine but
// usually need an explicit type argument; see "A note on type arguments" in
// the package doc. [WithEntryPoint] is bound to [Extism]; passing it to
// [Risor] or [Starlark] is a compile error.
func New[E Engine](src Source, opts ...Option[E]) (platform.Evaluator, error) {
	cfg := &config{}
	for _, o := range opts {
		if o != nil {
			o(cfg)
		}
	}
	var e E
	if _, ok := any(e).(Extism); ok && cfg.entryPoint == "" {
		return nil, fmt.Errorf("polyscript.New[Extism]: %w", extismMachine.ErrEntryPointRequired)
	}

	ldr, err := src.resolve()
	if err != nil {
		return nil, err
	}

	switch any(e).(type) {
	case Risor:
		return newRisor(ldr, cfg)
	case Starlark:
		return newStarlark(ldr, cfg)
	case Extism:
		return newExtism(ldr, cfg)
	default:
		return nil, fmt.Errorf("polyscript.New: unsupported engine type %T", e)
	}
}

func newRisor(ldr loader.Loader, cfg *config) (platform.Evaluator, error) {
	opts := []risorMachine.Option{risorMachine.WithLogHandler(cfg.handler)}
	if cfg.staticData != nil {
		opts = append(opts, risorMachine.WithStaticData(cfg.staticData))
	}
	return risorMachine.FromRisorLoader(ldr, opts...)
}

func newStarlark(ldr loader.Loader, cfg *config) (platform.Evaluator, error) {
	opts := []starlarkMachine.Option{starlarkMachine.WithLogHandler(cfg.handler)}
	if cfg.staticData != nil {
		opts = append(opts, starlarkMachine.WithStaticData(cfg.staticData))
	}
	return starlarkMachine.FromStarlarkLoader(ldr, opts...)
}

func newExtism(ldr loader.Loader, cfg *config) (platform.Evaluator, error) {
	opts := []extismMachine.Option{
		extismMachine.WithEntryPoint(cfg.entryPoint),
		extismMachine.WithLogHandler(cfg.handler),
	}
	if cfg.staticData != nil {
		opts = append(opts, extismMachine.WithStaticData(cfg.staticData))
	}
	return extismMachine.FromExtismLoader(ldr, opts...)
}
