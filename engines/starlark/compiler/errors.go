package compiler

import "errors"

var (
	ErrBytecodeNil        = errors.New("starlark bytecode is nil")
	ErrContentNil         = errors.New("starlark content is nil")
	ErrExecCreationFailed = errors.New("unable to create starlark executable")
	ErrValidationFailed   = errors.New("starlark script validation error")
)
