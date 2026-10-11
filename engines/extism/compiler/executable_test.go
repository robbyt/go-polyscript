package compiler

import (
	"context"
	"errors"
	"testing"
	"time"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/engines/extism/adapters"
	engineTypes "github.com/robbyt/go-polyscript/engines/types"
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
			assert.Equal(t, engineTypes.Extism, exe.EngineType())
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

		t.Run("EngineType", func(t *testing.T) {
			engineType := exe.EngineType()
			assert.Equal(t, engineTypes.Extism, engineType)
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
		mockPlugin.On("Close", ctx).Return(nil).Once()

		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)
		assert.False(t, exe.IsClosed())

		t.Run("first close", func(t *testing.T) {
			err := exe.Close(ctx)
			require.NoError(t, err)
			assert.True(t, exe.IsClosed())
		})

		t.Run("second close (no-op)", func(t *testing.T) {
			err := exe.Close(ctx)
			assert.NoError(t, err)
		})

		t.Run("getters return nil after close", func(t *testing.T) {
			assert.Nil(t, exe.GetByteCode())
			assert.Nil(t, exe.GetExtismByteCode())
		})

		t.Run("WithPlugin fails after close", func(t *testing.T) {
			called := false
			err := exe.WithPlugin(func(adapters.CompiledPlugin) error {
				called = true
				return nil
			})
			require.ErrorIs(t, err, ErrExecutableClosed)
			assert.False(t, called, "fn must not run on a closed executable")
		})

		mockPlugin.AssertExpectations(t)
	})

	t.Run("Close propagates plugin error once", func(t *testing.T) {
		ctx := t.Context()
		closeErr := errors.New("close kaboom")
		mockPlugin := new(MockCompiledPlugin)
		mockPlugin.On("Close", ctx).Return(closeErr).Once()

		exe := NewExecutable([]byte("mock wasm bytes"), mockPlugin, "run")
		require.NotNil(t, exe)

		require.ErrorIs(t, exe.Close(ctx), closeErr)
		assert.True(t, exe.IsClosed())
		require.NoError(t, exe.Close(ctx), "second Close must not retry the plugin Close")
		mockPlugin.AssertExpectations(t)
	})

	t.Run("Close on nil and zero-value executables", func(t *testing.T) {
		var nilExe *Executable
		require.NoError(t, nilExe.Close(t.Context()))

		zeroExe := &Executable{}
		require.NoError(t, zeroExe.Close(t.Context()))
		assert.True(t, zeroExe.IsClosed())
	})
}

func TestExecutable_WithPlugin(t *testing.T) {
	t.Parallel()

	wasmBytes := []byte("mock wasm bytes")
	entryPoint := "run"

	t.Run("passes the compiled plugin to fn", func(t *testing.T) {
		mockPlugin := new(MockCompiledPlugin)
		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)

		var got adapters.CompiledPlugin
		err := exe.WithPlugin(func(p adapters.CompiledPlugin) error {
			got = p
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, mockPlugin, got)
		mockPlugin.AssertExpectations(t)
	})

	t.Run("returns fn error", func(t *testing.T) {
		fnErr := errors.New("fn kaboom")
		exe := NewExecutable(wasmBytes, new(MockCompiledPlugin), entryPoint)
		require.NotNil(t, exe)

		err := exe.WithPlugin(func(adapters.CompiledPlugin) error { return fnErr })
		require.ErrorIs(t, err, fnErr)
	})

	t.Run("nil plugin", func(t *testing.T) {
		exe := &Executable{}
		err := exe.WithPlugin(func(adapters.CompiledPlugin) error {
			t.Error("fn must not run without a plugin")
			return nil
		})
		require.ErrorIs(t, err, ErrBytecodeNil)
	})

	t.Run("Close waits for in-flight users", func(t *testing.T) {
		pluginClosed := make(chan struct{})
		mockPlugin := new(MockCompiledPlugin)
		mockPlugin.On("Close", mock.Anything).Return(nil).Once().
			Run(func(mock.Arguments) { close(pluginClosed) })

		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)

		inUse := make(chan struct{})
		release := make(chan struct{})
		useDone := make(chan error, 1)
		go func() {
			useDone <- exe.WithPlugin(func(adapters.CompiledPlugin) error {
				close(inUse)
				<-release
				return nil
			})
		}()
		<-inUse

		closeDone := make(chan error, 1)
		go func() { closeDone <- exe.Close(t.Context()) }()

		select {
		case <-pluginClosed:
			t.Fatal("plugin closed while still in use")
		case <-time.After(50 * time.Millisecond):
		}

		close(release)
		require.NoError(t, <-useDone)
		require.NoError(t, <-closeDone)
		<-pluginClosed
		mockPlugin.AssertExpectations(t)
	})

	t.Run("concurrent users do not block each other", func(t *testing.T) {
		exe := NewExecutable(wasmBytes, new(MockCompiledPlugin), entryPoint)
		require.NotNil(t, exe)

		firstIn := make(chan struct{})
		secondIn := make(chan struct{})
		firstDone := make(chan error, 1)
		go func() {
			firstDone <- exe.WithPlugin(func(adapters.CompiledPlugin) error {
				close(firstIn)
				select {
				case <-secondIn:
					return nil
				case <-time.After(5 * time.Second):
					return errors.New("second WithPlugin never ran concurrently")
				}
			})
		}()
		<-firstIn

		err := exe.WithPlugin(func(adapters.CompiledPlugin) error {
			close(secondIn)
			return nil
		})
		require.NoError(t, err)
		require.NoError(t, <-firstDone)
	})
}
