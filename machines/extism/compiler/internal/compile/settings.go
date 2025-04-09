package compile

import (
	extismSDK "github.com/extism/go-sdk"
	"github.com/tetratelabs/wazero"
)

// Settings holds configuration for compiling a WASM module
type Settings struct {
	// EnableWASI enables WASI support in the plugin
	EnableWASI bool
	// RuntimeConfig allows customizing the wazero runtime configuration
	RuntimeConfig wazero.RuntimeConfig
	// HostFunctions are additional host functions to be registered with the plugin
	HostFunctions []extismSDK.HostFunction
}

// WithDefaultCompileSettings returns the default compilation options
func WithDefaultCompileSettings() *Settings {
	return &Settings{
		EnableWASI:    true,
		RuntimeConfig: wazero.NewRuntimeConfig(),
	}
}
