package starlark

import (
	machineTypes "github.com/robbyt/go-polyscript/machines/types"
	starlarkLib "go.starlark.net/starlark"
)

// executable represents a compiled Starlark script
type executable struct {
	scriptBodyBytes []byte
	ByteCode        *starlarkLib.Program
}

// Keep the existing constructor and methods
func newExecutable(scriptBodyBytes []byte, byteCode *starlarkLib.Program) *executable {
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

func (e *executable) GetStarlarkByteCode() *starlarkLib.Program {
	return e.ByteCode
}

func (e *executable) GetMachineType() machineTypes.Type {
	return machineTypes.Starlark
}
