package script

import (
	machineTypes "github.com/robbyt/go-polyscript/machines/types"
)

// ExecutableContent represents validated script content that is ready for execution or compilation.
// It provides access to the script's source code and its compiled bytecode.
// Implementations like [`risor.Executable`](internal/vm/cpu/risor/executable.go)
// store the script content and the compiled bytecode for execution.
type ExecutableContent interface {
	// GetSource returns the original script content as a string.
	// This is the source code before any compilation or execution.
	GetSource() string

	// GetByteCode returns the compiled bytecode of the script in a runtime-specific format.
	// This bytecode object is asserted into the type the target machine requires. If the
	// target machine is unable to assert the bytecode into the correct type, it will return
	// an error at runtime, so the MachineType and ByteCode must be compatible.
	GetByteCode() any

	// GetMachineType returns the machine type this script is intended to run on.
	GetMachineType() machineTypes.Type
}
