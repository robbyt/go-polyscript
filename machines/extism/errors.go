package extism

import "errors"

var (
	ErrContentNil         = errors.New("wasm content is nil")
	ErrValidationFailed   = errors.New("wasm script validation error")
	ErrBytecodeNil        = errors.New("wasm bytecode is nil")
	ErrNoInstructions     = errors.New("wasm bytecode has zero instructions")
	ErrExecCreationFailed = errors.New("unable to create wasm executable")
)
