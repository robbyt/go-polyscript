package compiler

import (
	"github.com/deepnoodle-ai/risor/v2/pkg/bytecode"
	machineTypes "github.com/robbyt/go-polyscript/engines/types"
)

type executable struct {
	scriptBodyBytes []byte
	ByteCode        *bytecode.Code
}

func newExecutable(scriptBodyBytes []byte, byteCode *bytecode.Code) *executable {
	if len(scriptBodyBytes) == 0 || byteCode == nil {
		return nil
	}

	return &executable{
		scriptBodyBytes: scriptBodyBytes,
		ByteCode:        byteCode,
	}
}

func (e *executable) GetSource() string {
	return string(e.scriptBodyBytes)
}

func (e *executable) GetByteCode() any {
	return e.ByteCode
}

func (e *executable) GetRisorByteCode() *bytecode.Code {
	return e.ByteCode
}

func (e *executable) GetMachineType() machineTypes.Type {
	return machineTypes.Risor
}
