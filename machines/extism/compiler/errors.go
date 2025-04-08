package compiler

import "errors"

var (
	ErrBytecodeNil        = errors.New("wasm bytecode is nil")
	ErrContentNil         = errors.New("wasm content is nil")
	ErrValidationFailed   = errors.New("wasm script validation error")
	ErrExecCreationFailed = errors.New("unable to create wasm executable")
)
