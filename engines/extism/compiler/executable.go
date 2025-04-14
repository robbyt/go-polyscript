package compiler

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/robbyt/go-polyscript/engines/extism/adapters"
	machineTypes "github.com/robbyt/go-polyscript/engines/types"
)

var ErrExecutableClosed = errors.New("executable is closed")

// Executable implements script.ExecutableContent for Extism WASM modules
type Executable struct {
	scriptBytes []byte
	ByteCode    adapters.CompiledPlugin
	entryPoint  string
	closed      atomic.Bool
	rwMutex     sync.RWMutex
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
		ByteCode:    byteCode,
		entryPoint:  entryPoint,
	}
}

// GetSource returns the original script content
func (e *Executable) GetSource() string {
	return string(e.scriptBytes)
}

// GetByteCode returns the compiled plugin as a generic interface
func (e *Executable) GetByteCode() any {
	e.rwMutex.RLock()
	defer e.rwMutex.RUnlock()
	return e.ByteCode
}

// GetExtismByteCode returns the compiled plugin with its proper type
func (e *Executable) GetExtismByteCode() adapters.CompiledPlugin {
	e.rwMutex.RLock()
	defer e.rwMutex.RUnlock()
	return e.ByteCode
}

// GetMachineType returns the Extism machine type
func (e *Executable) GetMachineType() machineTypes.Type {
	return machineTypes.Extism
}

// GetEntryPoint returns the name of the entry point function
func (e *Executable) GetEntryPoint() string {
	return e.entryPoint
}

// Close implements io.Closer
func (e *Executable) Close(ctx context.Context) error {
	e.rwMutex.Lock()
	defer e.rwMutex.Unlock()

	if e.closed.CompareAndSwap(false, true) {
		return e.ByteCode.Close(ctx)
	}
	return nil
}
