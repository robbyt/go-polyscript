package compiler

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/robbyt/go-polyscript/engines/extism/adapters"
	engineTypes "github.com/robbyt/go-polyscript/engines/types"
)

// ErrExecutableClosed is returned when a closed Executable's compiled plugin
// is used, e.g. by an Eval after the evaluator's Close.
var ErrExecutableClosed = errors.New("executable is closed")

// Executable implements script.ExecutableContent for Extism WASM modules.
// It owns the compiled plugin (and its wazero runtime) until Close.
type Executable struct {
	scriptBytes []byte
	byteCode    adapters.CompiledPlugin
	entryPoint  string

	// closing is set as soon as Close is called, before it waits for the
	// write lock, so new users fail fast instead of queueing behind a
	// pending Close. closed is set once the plugin has been released.
	closing atomic.Bool
	closed  atomic.Bool

	// rwMutex is held for reading while the plugin is in use (see
	// WithPlugin) and for writing by Close, so the runtime is never
	// released under an in-flight Eval.
	rwMutex sync.RWMutex
}

// NewExecutable creates a new Executable instance
func NewExecutable(
	scriptBytes []byte,
	byteCode adapters.CompiledPlugin,
	entryPoint string,
) *Executable {
	if len(scriptBytes) == 0 || byteCode == nil || entryPoint == "" {
		return nil
	}
	return &Executable{
		scriptBytes: scriptBytes,
		byteCode:    byteCode,
		entryPoint:  entryPoint,
	}
}

// GetSource returns the original script content
func (e *Executable) GetSource() string {
	return string(e.scriptBytes)
}

// GetByteCode returns the compiled plugin as a generic interface, or nil
// once Close has been called.
func (e *Executable) GetByteCode() any {
	return e.GetExtismByteCode()
}

// GetExtismByteCode returns the compiled plugin with its proper type, or nil
// once Close has been called. The returned plugin is not protected against a
// concurrent Close; use WithPlugin to run code against it safely.
func (e *Executable) GetExtismByteCode() adapters.CompiledPlugin {
	if e.closing.Load() {
		return nil
	}
	e.rwMutex.RLock()
	defer e.rwMutex.RUnlock()
	if e.closing.Load() {
		return nil
	}
	return e.byteCode
}

// WithPlugin calls fn with the compiled plugin while holding a read lock, so
// a concurrent Close waits for fn to return before releasing the runtime.
// Concurrent WithPlugin calls do not block each other, and once Close has
// been called new calls fail fast rather than waiting for it. It returns
// ErrExecutableClosed if Close was called, ErrBytecodeNil if there is no
// plugin, and otherwise fn's error. fn must not call Close or other methods
// of e that take its lock.
func (e *Executable) WithPlugin(fn func(plugin adapters.CompiledPlugin) error) error {
	// Checked before RLock: sync.RWMutex blocks new readers while a writer
	// waits, so without this a pending Close would stall every new Eval
	// until in-flight ones return.
	if e.closing.Load() {
		return ErrExecutableClosed
	}
	e.rwMutex.RLock()
	defer e.rwMutex.RUnlock()
	if e.closing.Load() {
		return ErrExecutableClosed
	}
	if e.byteCode == nil {
		return ErrBytecodeNil
	}
	return fn(e.byteCode)
}

// IsClosed reports whether Close has released the plugin. It stays false
// while a Close waits for in-flight WithPlugin calls.
func (e *Executable) IsClosed() bool {
	return e.closed.Load()
}

// EngineType returns the Extism engine type
func (e *Executable) EngineType() engineTypes.Type {
	return engineTypes.Extism
}

// GetEntryPoint returns the name of the entry point function
func (e *Executable) GetEntryPoint() string {
	return e.entryPoint
}

// Close releases the compiled plugin and its wazero runtime. New WithPlugin
// calls fail with ErrExecutableClosed as soon as Close is called, and Close
// waits for in-flight ones (i.e. running Evals) to return; cancel their
// contexts to stop long-running guests. Only the first call closes the
// plugin and returns its error; later calls wait for it and return nil. Safe
// on a nil Executable.
func (e *Executable) Close(ctx context.Context) error {
	if e == nil {
		return nil
	}
	e.closing.Store(true)
	e.rwMutex.Lock()
	defer e.rwMutex.Unlock()

	// The write lock serializes Close calls, so check-then-set is safe.
	if e.closed.Load() {
		return nil
	}
	defer e.closed.Store(true)
	if e.byteCode == nil {
		return nil
	}
	return e.byteCode.Close(ctx)
}
