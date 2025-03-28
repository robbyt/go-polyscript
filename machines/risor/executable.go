package risor

import (
	risorCompiler "github.com/risor-io/risor/compiler"
	machineTypes "github.com/robbyt/go-polyscript/machines/types"
)

type Executable struct {
	scriptBodyBytes []byte
	ByteCode        *risorCompiler.Code
}

func NewExecutable(scriptBodyBytes []byte, byteCode *risorCompiler.Code) *Executable {
	if len(scriptBodyBytes) == 0 || byteCode == nil {
		return nil
	}

	return &Executable{
		scriptBodyBytes: scriptBodyBytes,
		ByteCode:        byteCode,
	}
}

func (e *Executable) GetSource() string {
	return string(e.scriptBodyBytes)
}

func (e *Executable) GetByteCode() any {
	return e.ByteCode
}

func (e *Executable) GetRisorByteCode() *risorCompiler.Code {
	return e.ByteCode
}

func (e *Executable) GetMachineType() machineTypes.Type {
	return machineTypes.Risor
}
