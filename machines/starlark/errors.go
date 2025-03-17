package starlark

import "errors"

var ErrContentNil = errors.New("starlark content is nil")
var ErrValidationFailed = errors.New("starlark script validation error")
var ErrBytecodeNil = errors.New("starlark bytecode is nil")
var ErrNoInstructions = errors.New("starlark bytecode has zero instructions")
var ErrExecCreationFailed = errors.New("unable to create starlark executable")
