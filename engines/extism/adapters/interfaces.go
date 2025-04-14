package adapters

import (
	"context"

	extismSDK "github.com/extism/go-sdk"
)

type ExtismExecutable struct {
	GetExtismExecutable func() *extismSDK.CompiledPlugin
}

// CompiledPlugin is an interface for abstracting the extismSDK.CompiledPlugin
type CompiledPlugin interface {
	Instance(ctx context.Context, config extismSDK.PluginInstanceConfig) (PluginInstance, error)
	Close(ctx context.Context) error
}

// PluginInstance is an interface for abstracting the extismSDK.Plugin
type PluginInstance interface {
	Call(name string, data []byte) (uint32, []byte, error)
	CallWithContext(ctx context.Context, name string, data []byte) (uint32, []byte, error)
	FunctionExists(name string) bool
	Close(ctx context.Context) error
}
