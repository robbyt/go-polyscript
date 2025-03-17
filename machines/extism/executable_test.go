package extism

import (
	"context"
	"testing"

	extismSDK "github.com/extism/go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	machineTypes "github.com/robbyt/go-polyscript/machines/types"
)

// MockCompiledPlugin is a mock implementation of the compiledPlugin interface
type MockCompiledPlugin struct {
	mock.Mock
}

func (m *MockCompiledPlugin) Instance(ctx context.Context, config extismSDK.PluginInstanceConfig) (pluginInstance, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(pluginInstance), args.Error(1)
}

func (m *MockCompiledPlugin) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockPluginInstance is a mock implementation of the pluginInstance interface
type MockPluginInstance struct {
	mock.Mock
}

func (m *MockPluginInstance) Call(name string, data []byte) (uint32, []byte, error) {
	args := m.Called(name, data)
	return uint32(args.Int(0)), args.Get(1).([]byte), args.Error(2)
}

func (m *MockPluginInstance) CallWithContext(ctx context.Context, name string, data []byte) (uint32, []byte, error) {
	args := m.Called(ctx, name, data)
	return uint32(args.Int(0)), args.Get(1).([]byte), args.Error(2)
}

func (m *MockPluginInstance) FunctionExists(name string) bool {
	args := m.Called(name)
	return args.Bool(0)
}

func (m *MockPluginInstance) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestNewExecutable(t *testing.T) {
	t.Parallel()

	// Test data
	wasmBytes := []byte("mock wasm bytes")
	entryPoint := "run"

	// Create a mock plugin
	mockPlugin := new(MockCompiledPlugin)

	// Empty entry point test
	t.Run("empty entry point", func(t *testing.T) {
		exe := NewExecutable(wasmBytes, mockPlugin, "")
		assert.Nil(t, exe)
	})

	// Empty script bytes test
	t.Run("empty script bytes", func(t *testing.T) {
		exe := NewExecutable(nil, mockPlugin, entryPoint)
		assert.Nil(t, exe)
	})

	// Nil plugin test
	t.Run("nil plugin", func(t *testing.T) {
		exe := NewExecutable(wasmBytes, nil, entryPoint)
		assert.Nil(t, exe)
	})

	// Valid creation test
	t.Run("valid creation", func(t *testing.T) {
		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)

		// Verify properties
		assert.Equal(t, string(wasmBytes), exe.GetSource())
		assert.Equal(t, mockPlugin, exe.GetByteCode())
		assert.Equal(t, mockPlugin, exe.GetExtismByteCode())
		assert.Equal(t, machineTypes.Extism, exe.GetMachineType())
		assert.Equal(t, entryPoint, exe.GetEntryPoint())
		assert.False(t, exe.closed.Load())
	})
}

func TestExecutable_Close(t *testing.T) {
	t.Parallel()

	// Test data
	wasmBytes := []byte("mock wasm bytes")
	entryPoint := "run"
	ctx := context.Background()

	// Create and setup mock plugin
	mockPlugin := new(MockCompiledPlugin)
	mockPlugin.On("Close", ctx).Return(nil)

	// Create executable
	exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
	require.NotNil(t, exe)

	// Verify initial state
	assert.False(t, exe.closed.Load())

	// Close executable
	err := exe.Close(ctx)
	require.NoError(t, err)

	// Verify closed state
	assert.True(t, exe.closed.Load())

	// Verify idempotent close - should not call plugin Close again
	err = exe.Close(ctx)
	assert.NoError(t, err)

	// Verify expectations
	mockPlugin.AssertExpectations(t)
}
