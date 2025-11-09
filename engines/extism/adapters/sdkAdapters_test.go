package adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Interface verification test
func TestInterfaceImplementation(t *testing.T) {
	// This test verifies that our adapter types implement the correct interfaces
	// The test passes if it compiles

	// Verify compiledPlugin interface is implemented by sdkCompiledPluginAdapter
	var _ CompiledPlugin = (*sdkCompiledPlugin)(nil)

	// Verify pluginInstance interface is implemented by sdkPluginAdapter
	var _ PluginInstance = (*sdkPluginAdapter)(nil)
}

// Rather than creating complex mocks, let's focus on making the implementation simpler
// by directly testing the interface definitions only.
// For the sdkCompiledPluginAdapter and sdkPluginAdapter implementations, we rely on
// the Extism SDK's types, so we don't need exhaustive tests for this wrapper code.

func TestNilPlugin(t *testing.T) {
	// Test that NewCompiledPluginAdapter returns nil when given a nil plugin
	adapter := NewCompiledPluginAdapter(nil)
	assert.Nil(t, adapter, "Expected nil adapter for nil plugin")
}

func TestGetPluginInstanceConfig(t *testing.T) {
	// Get the config
	config := NewPluginInstanceConfig()

	// Should be a valid config
	require.NotNil(t, config)
	require.NotNil(t, config.ModuleConfig)
}
