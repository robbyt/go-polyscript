package risor

import (
	"context"
	"log/slog"

	"github.com/robbyt/go-polyscript/platform/data"
)

// Option configures [FromRisorLoader]. Apply in any order; later options
// override earlier ones for the same field.
type Option func(*config)

type config struct {
	handler      slog.Handler
	staticData   map[string]any
	dataProvider data.Provider
	// lifetime is held only while the constructor runs; it is handed to
	// the executable unit, which keeps just a func to stop its hook.
	lifetime context.Context
}

// WithLogHandler sets the slog.Handler used for diagnostic logging by the
// compiler and evaluator. A nil handler is permitted and means "inherit
// from slog.Default()" — equivalent to omitting the option.
func WithLogHandler(h slog.Handler) Option {
	return func(c *config) { c.handler = h }
}

// WithStaticData attaches a fixed map of values that the script will see
// alongside any runtime data added later via Evaluator.AddDataToContext.
//
// Combining [WithStaticData] with [WithDataProvider] is unsupported:
// [WithDataProvider] wins and the static data is ignored.
func WithStaticData(d map[string]any) Option {
	return func(c *config) { c.staticData = d }
}

// WithDataProvider supplies a custom [data.Provider]. If unset,
// [FromRisorLoader] builds a [data.ContextProvider] (or a
// [data.CompositeProvider] when [WithStaticData] is set).
//
// [WithDataProvider] takes precedence over [WithStaticData]; pass exactly
// one of them per call.
func WithDataProvider(p data.Provider) Option {
	return func(c *config) { c.dataProvider = p }
}

// WithLifetime sets the ctx that bounds the evaluator's lifetime, in place
// of the ctx passed to [FromRisorLoader]. A Risor evaluator holds nothing
// to release, so the end of its lifetime changes nothing and Eval keeps
// working; the option exists so that code written for every engine can pass
// it. If the ctx is already done when construction finishes,
// [FromRisorLoader] returns an error. A nil ctx means the constructor's ctx.
func WithLifetime(ctx context.Context) Option {
	return func(c *config) { c.lifetime = ctx }
}
