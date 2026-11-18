package evaluator

import (
	"errors"

	"github.com/robbyt/go-polyscript/engines/risor/internal"
)

// ErrUintOverflow is returned by Eval when the input data holds an unsigned
// integer above math.MaxInt64, which Risor's int64 cannot represent. Rejecting
// it avoids a silent wrap to a negative value (e.g. MaxUint64 reading as -1).
var ErrUintOverflow = internal.ErrUintOverflow

// ErrAsMapTypeMismatch is returned by (*execResult).AsMap when the underlying
// value is not a map[string]any. Wrapped with the actual Go type via "%T", so
// callers can use errors.Is(err, ErrAsMapTypeMismatch) and still read the
// concrete type from err.Error().
var ErrAsMapTypeMismatch = errors.New("risor: AsMap expected map[string]any")
