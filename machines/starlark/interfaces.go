package starlark

import starlarkLib "go.starlark.net/starlark"

type StarlarkExecutable struct {
	GetStarlarkByteCode func() *starlarkLib.Program
}
