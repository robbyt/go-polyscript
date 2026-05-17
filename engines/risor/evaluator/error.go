package evaluator

import (
	"errors"
)

// Error is returned from (*Evaluator).Eval for Risor execution failures —
// VM exec errors (compile/runtime crash) and script-side error/function
// returns. It implements error and exposes the diagnostic Inspect()
// output of the script-side error/function object via ScriptResult.
//
// Callers using `if err != nil` need no changes. Callers wanting the
// script-side diagnostic can either type-assert with errors.As or call
// GetErrorDetails(err).
type Error struct {
	// Msg is the human-readable wrapper message.
	Msg string

	// ScriptResult is the failing object's Inspect() output when the
	// script returned an error or function value. Empty for VM exec
	// failures and non-script failures.
	ScriptResult string

	// Cause is the wrapped VM/exec error, if any. nil for script-side
	// error/function returns.
	Cause error
}

func (e *Error) Error() string { return e.Msg }

func (e *Error) Unwrap() error { return e.Cause }

// GetErrorDetails returns the script-side diagnostic (the Inspect()
// output of the returned error or function object) from a Risor
// evaluator error, or "" if err isn't one of ours or carries no script
// detail. Convenience for callers who prefer not to declare a typed
// target for errors.As.
func GetErrorDetails(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.ScriptResult
	}
	return ""
}
