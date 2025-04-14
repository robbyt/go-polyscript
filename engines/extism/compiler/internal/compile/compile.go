package compile

import (
	"context"
	"encoding/base64"
	"fmt"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/engines/extism/adapters"
)

// CompileBase64 creates a compiled Extism plugin from base64-encoded WASM content
func CompileBase64(
	ctx context.Context,
	scriptContent string,
	opts *Settings,
) (adapters.CompiledPlugin, error) {
	wasmBytes, err := base64.StdEncoding.DecodeString(scriptContent)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidBinary, err)
	}
	return compile(ctx, wasmBytes, opts)
}

// CompileBytes creates a compiled Extism plugin from raw WASM bytes
func CompileBytes(
	ctx context.Context,
	wasmBytes []byte,
	opts *Settings,
) (adapters.CompiledPlugin, error) {
	return compile(ctx, wasmBytes, opts)
}

// compile creates a compiled Extism plugin from WASM bytes
func compile(
	ctx context.Context,
	wasmBytes []byte,
	opts *Settings,
) (adapters.CompiledPlugin, error) {
	if len(wasmBytes) == 0 {
		return nil, ErrContentNil
	}

	if opts == nil {
		opts = WithDefaultCompileSettings()
	}

	// Create manifest from wasm bytes
	manifest := extismSDK.Manifest{
		Wasm: []extismSDK.Wasm{
			extismSDK.WasmData{
				Data: wasmBytes,
			},
		},
	}

	// Configure the plugin
	config := extismSDK.PluginConfig{
		EnableWasi:    opts.EnableWASI,
		RuntimeConfig: opts.RuntimeConfig,
	}

	// Create compiled plugin using the SDK
	plugin, err := extismSDK.NewCompiledPlugin(ctx, manifest, config, opts.HostFunctions)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCompileFailed, err)
	}

	// Wrap the SDK plugin with our adapter
	return adapters.NewCompiledPluginAdapter(plugin), nil
}
