package extism

import (
	extismSDK "github.com/extism/go-sdk"
	"github.com/tetratelabs/wazero"
)

// compileOptions holds configuration for compiling a WASM module
type compileOptions struct {
	// EnableWASI enables WASI support in the plugin
	EnableWASI bool
	// RuntimeConfig allows customizing the wazero runtime configuration
	RuntimeConfig wazero.RuntimeConfig
	// HostFunctions are additional host functions to be registered with the plugin
	HostFunctions []extismSDK.HostFunction
}

// withDefaultCompileOptions returns the default compilation options
func withDefaultCompileOptions() *compileOptions {
	return &compileOptions{
		EnableWASI:    true,
		RuntimeConfig: wazero.NewRuntimeConfig(),
	}
}
