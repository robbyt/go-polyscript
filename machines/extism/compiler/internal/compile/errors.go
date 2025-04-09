package compile

import "errors"

var (
	ErrContentNil    = errors.New("wasm content is nil")
	ErrInvalidBinary = errors.New("invalid WASM binary (must be base64 encoded)")
	ErrCompileFailed = errors.New("failed to compile plugin")
)
