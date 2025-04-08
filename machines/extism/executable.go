package extism

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	machineTypes "github.com/robbyt/go-polyscript/machines/types"
)

var ErrExecutableClosed = errors.New("executable is closed")

// executable implements script.ExecutableContent for Extism WASM modules
type executable struct {
	scriptBytes []byte
	ByteCode    compiledPlugin
	entryPoint  string
	closed      atomic.Bool
	rwMutex     sync.RWMutex
}

// newExecutable creates a new Executable instance
func newExecutable(scriptBytes []byte, byteCode compiledPlugin, entryPoint string) *executable {
	if len(scriptBytes) == 0 || byteCode == nil || entryPoint == "" {
		return nil
	}
	return &executable{
		scriptBytes: scriptBytes,
		ByteCode:    byteCode,
		entryPoint:  entryPoint,
	}
}

// GetSource returns the original script content
func (e *executable) GetSource() string {
	return string(e.scriptBytes)
}

// GetByteCode returns the compiled plugin as a generic interface
func (e *executable) GetByteCode() any {
	e.rwMutex.RLock()
	defer e.rwMutex.RUnlock()
	return e.ByteCode
}

// GetExtismByteCode returns the compiled plugin with its proper type
func (e *executable) GetExtismByteCode() compiledPlugin {
	e.rwMutex.RLock()
	defer e.rwMutex.RUnlock()
	return e.ByteCode
}

// GetMachineType returns the Extism machine type
func (e *executable) GetMachineType() machineTypes.Type {
	return machineTypes.Extism
}

// GetEntryPoint returns the name of the entry point function
func (e *executable) GetEntryPoint() string {
	return e.entryPoint
}

// Close implements io.Closer
func (e *executable) Close(ctx context.Context) error {
	e.rwMutex.Lock()
	defer e.rwMutex.Unlock()

	if e.closed.CompareAndSwap(false, true) {
		return e.ByteCode.Close(ctx)
	}
	return nil
}
