package script

import "errors"

var ErrOldContent = errors.New("script content has not changed")
var ErrCompiler = errors.New("compiler failed or is invalid")
