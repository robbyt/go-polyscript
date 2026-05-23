// Package extism exposes the Extism (WebAssembly) engine through
// go-polyscript's loader/options surface.
//
// Most callers should use the higher-level generic constructor
// [polyscript.New] from the top-level package; this package is the
// engine-level building block beneath it.
//
// # Quick start
//
//	ldr, err := loader.NewFromBytes(wasmBytes)
//	if err != nil { return err }
//
//	eval, err := extism.FromExtismLoader(
//	    ldr,
//	    extism.WithEntryPoint("greet"),
//	    extism.WithStaticData(map[string]any{"input": "World"}),
//	)
//	if err != nil { return err }
//
//	result, err := eval.Eval(ctx)
//
// # Options
//
//   - [WithEntryPoint] — required; the exported WASM function to invoke
//   - [WithLogHandler] — slog.Handler for diagnostic logs (defaults to
//     slog.Default() when nil or omitted)
//   - [WithStaticData] — a fixed input map; layered with the runtime
//     ContextProvider via a CompositeProvider
//   - [WithDataProvider] — a custom [data.Provider] that bypasses the
//     ContextProvider/CompositeProvider composition; mutually exclusive
//     with [WithStaticData]
package extism

import (
	"context"
	"errors"
	"fmt"

	"github.com/robbyt/go-polyscript/engines/extism/compiler"
	"github.com/robbyt/go-polyscript/engines/extism/evaluator"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

// ErrEntryPointRequired is returned by [FromExtismLoader] when no entry
// point name was supplied via [WithEntryPoint].
var ErrEntryPointRequired = errors.New("extism: entry point is required (use WithEntryPoint)")

// FromExtismLoader builds an Extism (WASM) evaluator from a module loader.
//
// The constructor is the single public entry point for building an Extism
// evaluator from this package. Configure it with the With* options:
//
//	eval, err := extism.FromExtismLoader(
//	    ldr,
//	    extism.WithEntryPoint("greet"),
//	    extism.WithLogHandler(slog.Default().Handler()),
//	    extism.WithStaticData(map[string]any{"input": "World"}),
//	)
//
// [WithEntryPoint] is required and identifies the exported WASM function
// to invoke; [FromExtismLoader] returns [ErrEntryPointRequired] if it is
// missing.
//
// Logging falls back to [slog.Default] when [WithLogHandler] is omitted.
//
// If both [WithStaticData] and [WithDataProvider] are supplied,
// [WithDataProvider] takes precedence — pass exactly one of them per
// call to keep intent unambiguous.
func FromExtismLoader(ldr loader.Loader, opts ...Option) (*evaluator.Evaluator, error) {
	cfg := &config{}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	if cfg.entryPoint == "" {
		return nil, ErrEntryPointRequired
	}

	provider := resolveProvider(cfg)

	compilerOpts := []compiler.FunctionalOption{
		compiler.WithEntryPoint(cfg.entryPoint),
		compiler.WithLogHandler(cfg.handler),
	}
	comp, err := NewCompiler(compilerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Extism compiler: %w", err)
	}

	execUnitID := ""
	if u := ldr.GetSourceURL(); u != nil {
		execUnitID = u.String()
	}

	// FromExtismLoader is a one-shot startup constructor; compile uses a
	// fresh Background context. Callers needing cancellable compile
	// should drive script.NewExecutableUnit directly.
	execUnit, err := script.NewExecutableUnit(context.Background(), cfg.handler, execUnitID, ldr, comp, provider)
	if err != nil {
		return nil, err
	}

	return evaluator.New(
		cfg.handler,
		execUnit,
		evaluator.WithExitOutputMaxBytes(cfg.exitOutputMaxBytes),
	), nil
}

// NewCompiler creates a new Extism compiler using the functional options pattern.
// Returns a compiler implementing the script.Compiler interface.
func NewCompiler(opts ...compiler.FunctionalOption) (*compiler.Compiler, error) {
	return compiler.New(opts...)
}

// resolveProvider builds the data.Provider used by the evaluator. An
// explicit WithDataProvider wins; otherwise WithStaticData composes a
// StaticProvider with the standard ContextProvider; otherwise a bare
// ContextProvider.
func resolveProvider(cfg *config) data.Provider {
	if cfg.dataProvider != nil {
		return cfg.dataProvider
	}
	ctxProvider := data.NewContextProvider(constants.EvalData)
	if cfg.staticData == nil {
		return ctxProvider
	}
	return data.NewCompositeProvider(data.NewStaticProvider(cfg.staticData), ctxProvider)
}
