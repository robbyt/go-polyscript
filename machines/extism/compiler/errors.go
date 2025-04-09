package compiler

import "errors"

var (
	ErrBytecodeNil        = errors.New("wasm bytecode is nil")
	ErrContentNil         = errors.New("wasm content is nil")
	ErrExecCreationFailed = errors.New("unable to create wasm executable")
	ErrValidationFailed   = errors.New("wasm script validation error")
)
