package risor

import "errors"

var ErrContentNil = errors.New("content is nil")
var ErrValidationFailed = errors.New("risor script validation error")
var ErrBytecodeNil = errors.New("risor bytecode is nil")
var ErrNoInstructions = errors.New("risor bytecode has zero instructions")
var ErrExecCreationFailed = errors.New("unable to create risor executable")
