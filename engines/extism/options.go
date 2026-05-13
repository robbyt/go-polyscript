package extism

import (
	"log/slog"

	"github.com/robbyt/go-polyscript/platform/data"
)

// Option configures [FromExtismLoader]. Apply in any order; later options
// override earlier ones for the same field.
type Option func(*config)

type config struct {
	handler            slog.Handler
	staticData         map[string]any
	dataProvider       data.Provider
	entryPoint         string
	exitOutputMaxBytes int
}

// WithLogHandler sets the slog.Handler used for diagnostic logging by the
// compiler and evaluator. A nil handler is permitted and means "inherit
// from slog.Default()" — equivalent to omitting the option.
func WithLogHandler(h slog.Handler) Option {
	return func(c *config) { c.handler = h }
}

// WithStaticData attaches a fixed map of values that the WASM module will
// see alongside any runtime data added later via Evaluator.AddDataToContext.
//
// Combining [WithStaticData] with [WithDataProvider] is unsupported:
// [WithDataProvider] wins and the static data is ignored.
func WithStaticData(d map[string]any) Option {
	return func(c *config) { c.staticData = d }
}

// WithDataProvider supplies a custom [data.Provider]. If unset,
// [FromExtismLoader] builds a [data.ContextProvider] (or a
// [data.CompositeProvider] when [WithStaticData] is set).
//
// [WithDataProvider] takes precedence over [WithStaticData]; pass exactly
// one of them per call.
func WithDataProvider(p data.Provider) Option {
	return func(c *config) { c.dataProvider = p }
}

// WithEntryPoint sets the exported WASM function name to invoke. It is
// required; [FromExtismLoader] returns [ErrEntryPointRequired] if the
// option is omitted or the name is empty.
func WithEntryPoint(name string) Option {
	return func(c *config) { c.entryPoint = name }
}

// WithExitOutputMaxBytes caps the WASM-output snippet included in
// non-zero-exit-code error messages produced by the evaluator.
//
//   - zero (or omitted) → use the default cap (1024 bytes)
//   - positive          → truncate at that value; the original byte length
//                         is surfaced in the error so callers can tell
//                         something was elided
//   - negative          → no cap; the full output is included unchanged
//
// Mirrors the cap semantics of HTTPOptions.MaxBodySize.
func WithExitOutputMaxBytes(n int) Option {
	return func(c *config) { c.exitOutputMaxBytes = n }
}
