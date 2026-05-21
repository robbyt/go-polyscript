package evaluator

import "errors"

// ErrAsMapTypeMismatch is returned by (*execResult).AsMap when the underlying
// value is not a map[string]any. Wrapped with the actual Go type via "%T", so
// callers can use errors.Is(err, ErrAsMapTypeMismatch) and still read the
// concrete type from err.Error().
var ErrAsMapTypeMismatch = errors.New("risor: AsMap expected map[string]any")
