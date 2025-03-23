package starlark

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	starlarkLib "go.starlark.net/starlark"

	"github.com/robbyt/go-polyscript/execution/data"
)

// execResult is a wrapper around the starlark.Value interface
type execResult struct {
	starlarkLib.Value
	execTime    time.Duration
	scriptExeID string
	logHandler  slog.Handler
	logger      *slog.Logger
}

func newEvalResult(handler slog.Handler, obj starlarkLib.Value, execTime time.Duration, versionID string) *execResult {
	if handler == nil {
		defaultHandler := slog.NewTextHandler(os.Stdout, nil)
		handler = defaultHandler.WithGroup("starlark")
		// Create a logger from the handler rather than using slog directly
		defaultLogger := slog.New(handler)
		defaultLogger.Warn("Handler is nil, using the default logger configuration.")
	}

	if obj == nil {
		obj = starlarkLib.None
	}

	return &execResult{
		Value:       obj,
		execTime:    execTime,
		scriptExeID: versionID,
		logHandler:  handler,
		logger:      slog.New(handler.WithGroup("execResult")),
	}
}

func (r *execResult) String() string {
	return fmt.Sprintf(
		"ExecResult{Type: %s, Value: %v, ExecTime: %s, ScriptExeID: %s}",
		r.Type(), r.Value, r.GetExecTime(), r.GetScriptExeID())
}

func (r *execResult) Type() data.Types {
	// Map Starlark types to our internal types
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
	case "list":
		return data.LIST
	case "tuple":
		return data.TUPLE
	case "dict":
		return data.MAP
	case "set":
		return data.SET
	case "function":
		return data.FUNCTION
	default:
		r.logger.Error("Unknown type", "type", r.Value.Type())
		return data.ERROR
	}
}

func (r *execResult) GetScriptExeID() string {
	return r.scriptExeID
}

func (r *execResult) GetExecTime() string {
	return r.execTime.String()
}

func (r *execResult) Inspect() string {
	return r.Value.String()
}

// Interface returns the Go native type for the Starlark value
func (r *execResult) Interface() any {
	v, err := convertStarlarkValueToInterface(r.Value)
	if err != nil {
		r.logger.Error("Failed to convert Starlark value to interface", "error", err)
		return nil
	}
	return v
}
