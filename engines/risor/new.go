package risor

import (
	"fmt"

	"github.com/robbyt/go-polyscript/engines/risor/compiler"
	"github.com/robbyt/go-polyscript/engines/risor/evaluator"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

// FromRisorLoader builds a Risor evaluator from a script loader.
//
// The constructor is the single public entry point for building a Risor
// evaluator from this package. Configure it with the With* options:
//
//	eval, err := risor.FromRisorLoader(
//	    ldr,
//	    risor.WithLogHandler(slog.Default().Handler()),
//	    risor.WithStaticData(map[string]any{"version": "1.0.0"}),
//	)
//
// All options are optional. With no options the evaluator inherits the
// default slog handler (via [helpers.SetupLogger]) and uses a
// [data.ContextProvider] for runtime data.
//
// If both [WithStaticData] and [WithDataProvider] are supplied,
// [WithDataProvider] takes precedence — pass exactly one of them per
// call to keep intent unambiguous.
func FromRisorLoader(ldr loader.Loader, opts ...Option) (*evaluator.Evaluator, error) {
	cfg := &config{}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	provider := resolveProvider(cfg)

	compiler, err := NewCompiler(compiler.WithCtxGlobal())
	if err != nil {
		return nil, fmt.Errorf("failed to create Risor compiler: %w", err)
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

// NewCompiler creates a new Risor compiler using the functional options pattern.
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
