package evaluator

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/robbyt/go-polyscript/engines/starlark/internal"
	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/platform/data"
	starlarkLib "go.starlark.net/starlark"
)

// execResult is a wrapper around the starlark.Value interface
type execResult struct {
	starlarkLib.Value
	execTime    time.Duration
	scriptExeID string
	logHandler  slog.Handler
	logger      *slog.Logger
}

func newEvalResult(
	handler slog.Handler,
	obj starlarkLib.Value,
	execTime time.Duration,
	versionID string,
) *execResult {
	handler, logger := helpers.SetupLogger(handler, "starlark", "execResult")

	if obj == nil {
		obj = starlarkLib.None
	}

	return &execResult{
		Value:       obj,
		execTime:    execTime,
		scriptExeID: versionID,
		logHandler:  handler,
		logger:      logger,
	}
}

func (r *execResult) String() string {
	return fmt.Sprintf(
		"ExecResult{Type: %s, Value: %v, ExecTime: %s, ScriptExeID: %s}",
		r.Type(), r.Value, r.ExecTime(), r.ScriptExeID())
}

// Type maps the Starlark type name to a data.Types constant. A range reports
// data.LIST because Interface() returns it as []any. Types with no counterpart
// report data.UNKNOWN.
func (r *execResult) Type() data.Types {
	switch r.Value.Type() {
	case "NoneType":
		return data.NONE
	case "bool":
		return data.BOOL
	case "int":
		return data.INT
	case "float":
		return data.FLOAT
	case "string":
		return data.STRING
	case "bytes":
		return data.BYTES
	case "list", "range":
		return data.LIST
	case "tuple":
		return data.TUPLE
	case "dict":
		return data.MAP
	case "set":
		return data.SET
	case "function", "builtin_function_or_method":
		return data.FUNCTION
	default:
		r.logger.Error("Unknown type", "type", r.Value.Type())
		return data.UNKNOWN
	}
}

func (r *execResult) ScriptExeID() string {
	return r.scriptExeID
}

func (r *execResult) ExecTime() time.Duration {
	return r.execTime
}

func (r *execResult) AsMap() (map[string]any, error) {
	// Convert directly rather than via Interface(), which logs and drops the
	// error (e.g. ErrDictKeyCollision) and would misreport it as a type mismatch.
	v, err := internal.ConvertStarlarkValueToInterface(r.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result: %w", err)
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w, got %T", ErrAsMapTypeMismatch, v)
	}
	return m, nil
}

func (r *execResult) Inspect() string {
	return r.Value.String()
}

// Interface returns the Go native type for the Starlark value; see
// internal.ConvertStarlarkValueToInterface for the mapping. Values that cannot
// be converted (functions, for example) are logged and returned as nil.
func (r *execResult) Interface() any {
	v, err := internal.ConvertStarlarkValueToInterface(r.Value)
	if err != nil {
		r.logger.Error("Failed to convert Starlark value to interface", "error", err)
		return nil
	}
	return v
}
