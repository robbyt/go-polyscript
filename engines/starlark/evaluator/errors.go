package evaluator

import (
	"errors"

	"github.com/robbyt/go-polyscript/engines/starlark/internal"
)

// ErrDictKeyCollision is returned by (*execResult).AsMap when the result dict
// has two distinct keys that stringify to the same map key, e.g. {1: "a",
// "1": "b"}. Rejecting it avoids silently dropping one of the entries.
var ErrDictKeyCollision = internal.ErrDictKeyCollision

// ErrAsMapTypeMismatch is returned by (*execResult).AsMap when the underlying
// value is not a map[string]any. Wrapped with the actual Go type via "%T", so
// callers can use errors.Is(err, ErrAsMapTypeMismatch) and still read the
// concrete type from err.Error().
var ErrAsMapTypeMismatch = errors.New("starlark: AsMap expected map[string]any")
