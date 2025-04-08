// The adapters for the Extism SDK types that wrap the SDK's concrete types are defined here.
// (CompiledPlugin and Plugin), allowing our code to work with our internal interfaces
// instead of directly depending on the Extism SDK. This make mocking possible for tests, and
// insulates local code from SDK changes.
package adapters

import (
	"context"

	extismSDK "github.com/extism/go-sdk"
)

// SdkPluginInstanceConfig is an interface that wraps Extism extismSDK.PluginInstanceConfig
// This interface is used here to enable testing and to abstract the methods exposed by extism.
type SdkPluginInstanceConfig interface {
	CallWithContext(ctx context.Context, functionName string, input []byte) (uint32, []byte, error)
	Close(ctx context.Context) error
}

// sdkCompiledPlugin adapts extismSDK.CompiledPlugin to our compiledPlugin interface
type sdkCompiledPlugin struct {
	plugin *extismSDK.CompiledPlugin
}

// NewCompiledPluginAdapter creates a new adapter for extismSDK.CompiledPlugin
func NewCompiledPluginAdapter(plugin *extismSDK.CompiledPlugin) CompiledPlugin {
	if plugin == nil {
		return nil
	}

	return &sdkCompiledPlugin{
		plugin: plugin,
	}
}

// Instance creates a new instance of the plugin
func (a *sdkCompiledPlugin) Instance(
	ctx context.Context,
	config extismSDK.PluginInstanceConfig,
) (PluginInstance, error) {
	instance, err := a.plugin.Instance(ctx, config)
	if err != nil {
		return nil, err
	}
	return &sdkPluginAdapter{instance: instance}, nil
}

// Close releases resources associated with the plugin
func (a *sdkCompiledPlugin) Close(ctx context.Context) error {
	return a.plugin.Close(ctx)
}

// sdkPluginAdapter adapts the Extism plugin instance to our pluginInstance interface
type sdkPluginAdapter struct {
	instance *extismSDK.Plugin
}

// CallWithContext calls a function in the plugin
func (a *sdkPluginAdapter) CallWithContext(
	ctx context.Context,
	name string,
	data []byte,
) (uint32, []byte, error) {
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
