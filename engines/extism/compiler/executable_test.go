package compiler

import (
	"context"
	"testing"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/engines/extism/adapters"
	machineTypes "github.com/robbyt/go-polyscript/engines/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockCompiledPlugin is a mock implementation of the compiledPlugin interface
type MockCompiledPlugin struct {
	mock.Mock
}

func (m *MockCompiledPlugin) Instance(
	ctx context.Context,
	config extismSDK.PluginInstanceConfig,
) (adapters.PluginInstance, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(adapters.PluginInstance), args.Error(1)
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

func (m *MockPluginInstance) CallWithContext(
	ctx context.Context,
	name string,
	data []byte,
) (uint32, []byte, error) {
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

// TestExecutable tests the functionality of Executable
func TestExecutable(t *testing.T) {
	t.Parallel()

	// Test creation scenarios
	t.Run("Creation", func(t *testing.T) {
		// Test data
		wasmBytes := []byte("mock wasm bytes")
		entryPoint := "run"

		// Create a mock plugin
		mockPlugin := new(MockCompiledPlugin)

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

		t.Run("empty entry point", func(t *testing.T) {
			exe := NewExecutable(wasmBytes, mockPlugin, "")
			assert.Nil(t, exe)
		})

		t.Run("empty script bytes", func(t *testing.T) {
			exe := NewExecutable(nil, mockPlugin, entryPoint)
			assert.Nil(t, exe)
		})

		t.Run("nil plugin", func(t *testing.T) {
			exe := NewExecutable(wasmBytes, nil, entryPoint)
			assert.Nil(t, exe)
		})
	})

	// Test getters
	t.Run("Getters", func(t *testing.T) {
		wasmBytes := []byte("mock wasm bytes")
		entryPoint := "run"
		mockPlugin := new(MockCompiledPlugin)

		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)

		t.Run("GetSource", func(t *testing.T) {
			source := exe.GetSource()
			assert.Equal(t, string(wasmBytes), source)
		})

		t.Run("GetByteCode", func(t *testing.T) {
			bytecode := exe.GetByteCode()
			assert.Equal(t, mockPlugin, bytecode)
		})

		t.Run("GetExtismByteCode", func(t *testing.T) {
			bytecode := exe.GetExtismByteCode()
			assert.Equal(t, mockPlugin, bytecode)
		})

		t.Run("GetMachineType", func(t *testing.T) {
			machineType := exe.GetMachineType()
			assert.Equal(t, machineTypes.Extism, machineType)
		})

		t.Run("GetEntryPoint", func(t *testing.T) {
			ep := exe.GetEntryPoint()
			assert.Equal(t, entryPoint, ep)
		})
	})

	// Test Close functionality (specific to Extism)
	t.Run("Close", func(t *testing.T) {
		ctx := t.Context()
		wasmBytes := []byte("mock wasm bytes")
		entryPoint := "run"

		mockPlugin := new(MockCompiledPlugin)
		mockPlugin.On("Close", ctx).Return(nil)

		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)
		assert.False(t, exe.closed.Load())

		t.Run("first close", func(t *testing.T) {
			err := exe.Close(ctx)
			require.NoError(t, err)
			assert.True(t, exe.closed.Load())
		})

		t.Run("second close (no-op)", func(t *testing.T) {
			err := exe.Close(ctx)
			assert.NoError(t, err)
		})

		mockPlugin.AssertExpectations(t)
	})
}
