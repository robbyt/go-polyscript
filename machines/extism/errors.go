package extism

import "errors"

var ErrContentNil = errors.New("wasm content is nil")
var ErrValidationFailed = errors.New("wasm script validation error")
var ErrBytecodeNil = errors.New("wasm bytecode is nil")
var ErrNoInstructions = errors.New("wasm bytecode has zero instructions")
var ErrExecCreationFailed = errors.New("unable to create wasm executable")
