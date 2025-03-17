package risor

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	risorObject "github.com/risor-io/risor/object"

	"github.com/robbyt/go-polyscript/execution/data"
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

func newEvalResult(handler slog.Handler, obj risorObject.Object, execTime time.Duration, versionID string) *execResult {
	if handler == nil {
		defaultHandler := slog.NewTextHandler(os.Stdout, nil)
		handler = defaultHandler.WithGroup("risor")
		// Create a logger from the handler rather than using slog directly
		defaultLogger := slog.New(handler)
		defaultLogger.Warn("Handler is nil, using the default logger configuration.")
	}

	return &execResult{
		Object:      obj,
		execTime:    execTime,
		scriptExeID: versionID,
		logHandler:  handler,
		logger:      slog.New(handler.WithGroup("execResult")),
	}
}

func (r *execResult) String() string {
	return fmt.Sprintf(
		"ExecResult{Type: %s, Value: %v, ExecTime: %s, ScriptExeID: %s}",
		r.Type(), r.Object, r.GetExecTime(), r.GetScriptExeID())
}

func (r *execResult) Type() data.Types {
	return data.Types(r.Object.Type())
}

func (r *execResult) GetScriptExeID() string {
	return r.scriptExeID
}

func (r *execResult) GetExecTime() string {
	return r.execTime.String()
}
