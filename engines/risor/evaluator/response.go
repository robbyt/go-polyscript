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
