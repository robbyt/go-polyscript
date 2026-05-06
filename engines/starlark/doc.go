// Package starlark exposes the Starlark configuration engine through
// go-polyscript's loader/options surface.
//
// Most callers should use the higher-level generic constructor
// [polyscript.New] from the top-level package; this package is the
// engine-level building block beneath it.
//
// # Quick start
//
//	ldr, err := loader.NewFromString(`_ = "hello"`)
//	if err != nil { return err }
//
//	eval, err := starlark.FromStarlarkLoader(
//	    ldr,
//	    starlark.WithStaticData(map[string]any{"name": "World"}),
//	)
//	if err != nil { return err }
//
//	result, err := eval.Eval(ctx)
//
// # Options
//
//   - [WithLogHandler] — slog.Handler for diagnostic logs (defaults to
//     slog.Default() when nil or omitted)
//   - [WithStaticData] — a fixed input map; layered with the runtime
//     ContextProvider via a CompositeProvider
//   - [WithDataProvider] — a custom [data.Provider] that bypasses the
//     ContextProvider/CompositeProvider composition; mutually exclusive
//     with [WithStaticData]
package starlark
