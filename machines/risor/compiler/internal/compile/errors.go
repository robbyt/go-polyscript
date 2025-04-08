package compile

import "errors"

var (
	ErrContentNil    = errors.New("script content is nil")
	ErrCompileFailed = errors.New("compilation failed")
)
