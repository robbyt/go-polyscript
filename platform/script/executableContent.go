package script

import (
	engineTypes "github.com/robbyt/go-polyscript/engines/types"
)

// ExecutableContent represents validated script content that is ready for execution or compilation.
// It provides access to the script's source code and its compiled bytecode.
// Implementations like [`risor.Executable`](../../engines/risor/compiler/executable.go)
// store the script content and the compiled bytecode for execution.
//
// Content that holds resources may also implement Close(context.Context)
// error to release them. Like [io.Closer], its behavior after the first call
// need not be defined: an [ExecutableUnit] calls it at most once, however
// many times or ways the unit itself is closed.
//
// A unit closes its content only through Close or the end of its lifetime
// (see [ExecutableUnit.BindLifetime]), never when the unit is garbage
// collected. The end of the lifetime can still close the content until a
// dropped unit has been collected; after that, a caller that retained the
// content from GetContent keeps using it and owns its release. The value
// GetByteCode returns need not keep the content alive, so a caller should
// retain the content, not just its bytecode. Content that holds resources
// should release them from its own cleanup (see [runtime.AddCleanup]) when
// it becomes unreachable without being closed, as [os.File] does.
type ExecutableContent interface {
	// GetSource returns the original script content as a string.
	// This is the source code before any compilation or execution.
	GetSource() string

	// GetByteCode returns the compiled bytecode of the script in a runtime-specific format.
	// This bytecode object is asserted into the type the target engine requires. If the
	// target engine is unable to assert the bytecode into the correct type, it will return
	// an error at runtime, so the engine type and ByteCode must be compatible.
	GetByteCode() any

	// EngineType returns the engine type this script is intended to run on.
	EngineType() engineTypes.Type
}
