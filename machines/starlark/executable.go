package starlark

import (
	starlarkLib "go.starlark.net/starlark"

	machineTypes "github.com/robbyt/go-polyscript/machines/types"
)

// Executable represents a compiled Starlark script
type Executable struct {
	scriptBodyBytes []byte
	ByteCode        *starlarkLib.Program
}

// Keep the existing constructor and methods
func NewExecutable(scriptBodyBytes []byte, byteCode *starlarkLib.Program) *Executable {
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

func (e *Executable) GetStarlarkByteCode() *starlarkLib.Program {
	return e.ByteCode
}

func (e *Executable) GetMachineType() machineTypes.Type {
	return machineTypes.Starlark
}
