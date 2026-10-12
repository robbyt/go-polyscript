package script

import (
	engineTypes "github.com/robbyt/go-polyscript/engines/types"
)

// ExecutableContent represents validated script content that is ready for execution or compilation.
// It provides access to the script's source code and its compiled bytecode.
// Each engine's compiler returns its own implementation, which stores the
// script content and the compiled bytecode for execution.
//
// Content that holds resources may also implement Close(context.Context)
// error to release them. Close must be safe to call more than once and
// concurrently and must release the resources once. Like
// [net/http.Server.Shutdown], each call's ctx bounds only that caller's wait
// for in-flight work, not the release itself.
//
// Such content may also implement WatchLifetime(done <-chan struct{}) to
// close itself once done is closed; see [ExecutableUnit.BindLifetime].
// Content that implements neither holds nothing to release. Nothing closes
// content when it is garbage collected: whoever owns it calls Close, or lets
// its lifetime end.
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
