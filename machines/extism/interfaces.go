package extism

import (
	"context"

	extismSDK "github.com/extism/go-sdk"
)

type ExtismExecutable struct {
	GetExtismExecutable func() *extismSDK.CompiledPlugin
}

// compiledPlugin is an interface for abstracting the extismSDK.CompiledPlugin
type compiledPlugin interface {
	Instance(ctx context.Context, config extismSDK.PluginInstanceConfig) (pluginInstance, error)
	Close(ctx context.Context) error
}

// pluginInstance is an interface for abstracting the extismSDK.Plugin
type pluginInstance interface {
	Call(name string, data []byte) (uint32, []byte, error)
	CallWithContext(ctx context.Context, name string, data []byte) (uint32, []byte, error)
	FunctionExists(name string) bool
	Close(ctx context.Context) error
}
