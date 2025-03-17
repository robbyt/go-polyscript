package extism

import (
	"context"

	extismSDK "github.com/extism/go-sdk"
)

// This file implements adapters for the Extism SDK types.
//
// It provides:
// - sdkCompiledPluginAdapter: Adapts extismSDK.CompiledPlugin to our compiledPlugin interface
// - sdkPluginAdapter: Adapts extismSDK.Plugin to our pluginInstance interface
//
// These adapters wrap the SDK's concrete types (CompiledPlugin and Plugin), allowing
// our code to work with our internal interfaces instead of directly depending on the
// Extism SDK. This makes testing easier and insulates our code from SDK changes.

// sdkCompiledPluginAdapter adapts extismSDK.CompiledPlugin to our compiledPlugin interface
type sdkCompiledPluginAdapter struct {
	plugin *extismSDK.CompiledPlugin
}

// newCompiledPluginAdapter creates a new adapter for extismSDK.CompiledPlugin
func newCompiledPluginAdapter(plugin *extismSDK.CompiledPlugin) compiledPlugin {
	return &sdkCompiledPluginAdapter{
		plugin: plugin,
	}
}

// Instance creates a new instance of the plugin
func (a *sdkCompiledPluginAdapter) Instance(ctx context.Context, config extismSDK.PluginInstanceConfig) (pluginInstance, error) {
	instance, err := a.plugin.Instance(ctx, config)
	if err != nil {
		return nil, err
	}
	return &sdkPluginAdapter{instance: instance}, nil
}

// Close releases resources associated with the plugin
func (a *sdkCompiledPluginAdapter) Close(ctx context.Context) error {
	return a.plugin.Close(ctx)
}

// sdkPluginAdapter adapts the Extism plugin instance to our pluginInstance interface
type sdkPluginAdapter struct {
	instance *extismSDK.Plugin
}

// CallWithContext calls a function in the plugin
func (a *sdkPluginAdapter) CallWithContext(ctx context.Context, name string, data []byte) (uint32, []byte, error) {
	return a.instance.CallWithContext(ctx, name, data)
}

// Call calls a function in the plugin without context
func (a *sdkPluginAdapter) Call(name string, data []byte) (uint32, []byte, error) {
	return a.instance.Call(name, data)
}

// FunctionExists checks if a function exists in the plugin
func (a *sdkPluginAdapter) FunctionExists(name string) bool {
	return a.instance.FunctionExists(name)
}

// Close releases resources associated with the instance
func (a *sdkPluginAdapter) Close(ctx context.Context) error {
	return a.instance.Close(ctx)
}
