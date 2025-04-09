package compile

import "errors"

var (
	ErrCompileFailed = errors.New("failed to compile starlark script")
	ErrContentNil    = errors.New("starlark content is nil")
)
