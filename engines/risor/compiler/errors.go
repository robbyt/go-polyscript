package compiler

import "errors"

var (
	ErrBytecodeNil        = errors.New("risor bytecode is nil")
	ErrContentNil         = errors.New("risor content is nil")
	ErrExecCreationFailed = errors.New("unable to create risor executable")
	ErrNoInstructions     = errors.New("risor bytecode has zero instructions")
	ErrValidationFailed   = errors.New("risor script validation error")
)
