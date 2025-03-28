package script

import "errors"

var (
	ErrOldContent = errors.New("script content has not changed")
	ErrCompiler   = errors.New("compiler failed or is invalid")
)
