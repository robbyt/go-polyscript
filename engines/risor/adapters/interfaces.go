package adapters

import "github.com/deepnoodle-ai/risor/v2/pkg/bytecode"

type RisorExecutable struct {
	GetRisorByteCode func() *bytecode.Code
}
