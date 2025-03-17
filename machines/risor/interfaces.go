package risor

import risorCompiler "github.com/risor-io/risor/compiler"

type RisorExecutable struct {
	GetRisorByteCode func() *risorCompiler.Code
}
