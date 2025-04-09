package compiler

import "errors"

var (
	ErrContentNil         = errors.New("starlark content is nil")
	ErrValidationFailed   = errors.New("starlark script validation error")
	ErrBytecodeNil        = errors.New("starlark bytecode is nil")
	ErrNoInstructions     = errors.New("starlark bytecode has zero instructions")
	ErrExecCreationFailed = errors.New("unable to create starlark executable")
)
