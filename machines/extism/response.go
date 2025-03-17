package extism

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/robbyt/go-polyscript/execution/data"
)

// execResult is a wrapper around the WASM execution result
type execResult struct {
	value       any
	execTime    time.Duration
	scriptExeID string
	logHandler  slog.Handler
	logger      *slog.Logger
}

func newEvalResult(handler slog.Handler, value any, execTime time.Duration, scriptExeID string) *execResult {
	if handler == nil {
		defaultHandler := slog.NewTextHandler(os.Stdout, nil)
		handler = defaultHandler.WithGroup("extism")
		// Create a logger from the handler rather than using slog directly
		defaultLogger := slog.New(handler)
		defaultLogger.Warn("Handler is nil, using the default logger configuration.")
	}

	return &execResult{
		value:       value,
		execTime:    execTime,
		scriptExeID: scriptExeID,
		logHandler:  handler,
		logger:      slog.New(handler.WithGroup("execResult")),
	}
}

func (r *execResult) getLogger() *slog.Logger {
	return r.logger
}

func (r *execResult) String() string {
	return fmt.Sprintf(
		"execResult{Type: %s, Value: %v, ExecTime: %s, ScriptExeID: %s}",
		r.Type(), r.value, r.GetExecTime(), r.GetScriptExeID(),
	)
}

func (r *execResult) Type() data.Types {
	// Map WASM result types to our internal types
	switch v := r.value.(type) {
	case nil:
		return data.NONE
	case bool:
		return data.BOOL
	case int32, int64, uint32, uint64:
		return data.INT
	case float32, float64:
		return data.FLOAT
	case string:
		return data.STRING
	case []any:
		return data.LIST
	case map[string]any:
		return data.MAP
	default:
		r.getLogger().Error("Unknown type", "type", fmt.Sprintf("%T", v))
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
	switch r.Type() {
	case data.MAP:
		// For map types, return JSON instead of Go's string representation
		jsonBytes, err := json.Marshal(r.value)
		if err == nil {
			return string(jsonBytes)
		}
		// If JSON marshaling fails, log and fallback to default
		r.getLogger().Error("Failed to marshal map to JSON", "error", err)
	}

	// Default string representation for other types
	return fmt.Sprintf("%v", r.value)
}

func (r *execResult) Interface() any {
	return r.value
}
