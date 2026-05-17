package evaluator

import (
	"errors"

	starlarkLib "go.starlark.net/starlark"
)

// Error is returned from (*Evaluator).Eval for Starlark execution failures
// — script-side fail() calls, runtime exceptions, and auto-called callable
// failures. It implements error and exposes the underlying
// *starlark.EvalError via Unwrap() and via the GetErrorDetails helper.
//
// Callers using `if err != nil` need no changes. Callers wanting access to
// the call stack or backtrace can either type-assert with errors.As or
// call GetErrorDetails(err).
type Error struct {
	// Msg is the human-readable wrapper message.
	Msg string

	// EvalErr is the raw Starlark error, or nil if the failure didn't
	// originate inside Starlark script execution.
	EvalErr *starlarkLib.EvalError
}

func (e *Error) Error() string { return e.Msg }

func (e *Error) Unwrap() error {
	if e.EvalErr == nil {
		return nil
	}
	return e.EvalErr
}

// GetErrorDetails returns the underlying *starlark.EvalError from a
// Starlark evaluator error, or nil if err isn't one of ours or carries no
// detail. Convenience for callers who prefer not to declare a typed target
// for errors.As.
func GetErrorDetails(err error) *starlarkLib.EvalError {
	var e *Error
	if errors.As(err, &e) {
		return e.EvalErr
	}
	return nil
}
