package compiler

import (
	risorCompiler "github.com/risor-io/risor/compiler"
	machineTypes "github.com/robbyt/go-polyscript/machines/types"
)

type executable struct {
	scriptBodyBytes []byte
	ByteCode        *risorCompiler.Code
}

func newExecutable(scriptBodyBytes []byte, byteCode *risorCompiler.Code) *executable {
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

func (e *executable) GetRisorByteCode() *risorCompiler.Code {
	return e.ByteCode
}

func (e *executable) GetMachineType() machineTypes.Type {
	return machineTypes.Risor
}
