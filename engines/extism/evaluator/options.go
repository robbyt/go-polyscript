package evaluator

// Option configures an [Evaluator]. Pass to [New].
type Option func(*Evaluator)

// WithExitOutputMaxBytes caps the WASM-output snippet included in non-zero-
// exit-code error messages. A zero value (or unset) uses
// [defaultExitOutputMaxBytes] (1024); a negative value disables the cap so
// the full output is included unchanged.
//
// This mirrors the cap semantics of HTTPOptions.MaxBodySize from the
// platform script loader.
func WithExitOutputMaxBytes(n int) Option {
	return func(e *Evaluator) { e.exitOutputMaxBytes = n }
}
