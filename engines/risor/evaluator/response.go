package evaluator

import (
	"fmt"
	"log/slog"
	"time"

	risorObject "github.com/deepnoodle-ai/risor/v2/pkg/object"
	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/platform/data"
)

// execResult is a wrapper around the risor object.Object interface, but with some features removed
// and return types adjusted to be generic.
type execResult struct {
	risorObject.Object
	execTime    time.Duration
	scriptExeID string
	logHandler  slog.Handler
	logger      *slog.Logger
}

func newEvalResult(
	handler slog.Handler,
	obj risorObject.Object,
	execTime time.Duration,
	versionID string,
) *execResult {
	handler, logger := helpers.SetupLogger(handler, "risor", "execResult")

	return &execResult{
		Object:      obj,
		execTime:    execTime,
		scriptExeID: versionID,
		logHandler:  handler,
		logger:      logger,
	}
}

func (r *execResult) String() string {
	return fmt.Sprintf(
		"ExecResult{Type: %s, Value: %v, ExecTime: %s, ScriptExeID: %s}",
		r.Type(), r.Object, r.ExecTime(), r.ScriptExeID())
}

// Type maps the Risor object type to a data.Types constant. Ranges and
// iterators report data.LIST because Interface() returns them as []any, and
// every callable kind (closure, builtin, go_func, partial) reports
// data.FUNCTION. Risor types with no data.Types counterpart (time, module,
// go_struct, and types defined by Risor modules) fall back to data.ERROR, the
// same fallback the other engines use for unknown types; Interface() still
// returns Risor's own Go conversion of those values (time.Time for a time).
func (r *execResult) Type() data.Types {
	if r.Object == nil {
		return data.NONE
	}

	switch t := r.Object.Type(); t {
	case risorObject.NIL:
		return data.NONE
	case risorObject.BOOL:
		return data.BOOL
	case risorObject.INT, risorObject.BYTE:
		return data.INT
	case risorObject.FLOAT:
		return data.FLOAT
	case risorObject.STRING:
		return data.STRING
	case risorObject.BYTES:
		return data.BYTES
	case risorObject.LIST, risorObject.RANGE, risorObject.ITER:
		return data.LIST
	case risorObject.MAP:
		return data.MAP
	case risorObject.FUNCTION, risorObject.BUILTIN, risorObject.GOFUNC, risorObject.PARTIAL:
		return data.FUNCTION
	case risorObject.ERROR:
		return data.ERROR
	default:
		r.logger.Error("Unknown type", "type", string(t))
		return data.ERROR
	}
}

// Interface returns the Go value Risor converts the result to. A range is
// expanded to []any (Risor returns []int64) so it matches the []any every
// other data.LIST result uses.
func (r *execResult) Interface() any {
	if r.Object == nil {
		return nil
	}

	if rng, ok := r.Object.(*risorObject.Range); ok {
		ints, _ := rng.Interface().([]int64)
		list := make([]any, len(ints))
		for i, n := range ints {
			list[i] = n
		}
		return list
	}

	return r.Object.Interface()
}

func (r *execResult) ScriptExeID() string {
	return r.scriptExeID
}

func (r *execResult) ExecTime() time.Duration {
	return r.execTime
}

func (r *execResult) AsMap() (map[string]any, error) {
	v := r.Interface()
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w, got %T", ErrAsMapTypeMismatch, v)
	}
	return m, nil
}
