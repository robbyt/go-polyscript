package compile

import "errors"

var (
	ErrCompileFailed = errors.New("failed to load wasm binary")
	ErrContentNil    = errors.New("wasm content is nil")
	ErrInvalidBinary = errors.New("invalid WASM binary (must be base64 encoded)")
)
