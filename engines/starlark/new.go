package starlark

import (
	"fmt"

	"github.com/robbyt/go-polyscript/engines/starlark/compiler"
	"github.com/robbyt/go-polyscript/engines/starlark/evaluator"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

// FromStarlarkLoader builds a Starlark evaluator from a script loader.
//
// The constructor is the single public entry point for building a Starlark
// evaluator from this package. Configure it with the With* options:
//
//	eval, err := starlark.FromStarlarkLoader(
//	    ldr,
//	    starlark.WithLogHandler(slog.Default().Handler()),
//	    starlark.WithStaticData(map[string]any{"version": "1.0.0"}),
//	)
//
// All options are optional. With no options the evaluator inherits the
// default slog handler (via [helpers.SetupLogger]) and uses a
// [data.ContextProvider] for runtime data.
//
// If both [WithStaticData] and [WithDataProvider] are supplied,
// [WithDataProvider] takes precedence — pass exactly one of them per
// call to keep intent unambiguous.
func FromStarlarkLoader(ldr loader.Loader, opts ...Option) (*evaluator.Evaluator, error) {
	cfg := &config{}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	provider := resolveProvider(cfg)

	compiler, err := NewCompiler(compiler.WithCtxGlobal())
	if err != nil {
		return nil, fmt.Errorf("failed to create Starlark compiler: %w", err)
	}

	execUnitID := ""
	if u := ldr.GetSourceURL(); u != nil {
		execUnitID = u.String()
	}

	execUnit, err := script.NewExecutableUnit(cfg.handler, execUnitID, ldr, compiler, provider)
	if err != nil {
		return nil, err
	}

	return evaluator.New(cfg.handler, execUnit), nil
}

// NewCompiler creates a new Starlark compiler using the functional options pattern.
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
