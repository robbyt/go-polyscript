package compile

import "errors"

var (
	ErrCompileFailed = errors.New("failed to compile risor script")
	ErrContentNil    = errors.New("risor content is nil")
)
