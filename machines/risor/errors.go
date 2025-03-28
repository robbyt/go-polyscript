package risor

import "errors"

var (
	ErrContentNil         = errors.New("content is nil")
	ErrValidationFailed   = errors.New("risor script validation error")
	ErrBytecodeNil        = errors.New("risor bytecode is nil")
	ErrNoInstructions     = errors.New("risor bytecode has zero instructions")
	ErrExecCreationFailed = errors.New("unable to create risor executable")
)
