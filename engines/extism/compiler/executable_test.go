package compiler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
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
			assert.False(t, exe.IsClosed())
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
		mockPlugin.On("Close", liveCtx()).Return(nil).Once()

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
			err := exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error {
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
		mockPlugin.On("Close", liveCtx()).Return(closeErr).Once()

		exe := NewExecutable([]byte("mock wasm bytes"), mockPlugin, "run")
		require.NotNil(t, exe)

		require.ErrorIs(t, exe.Close(ctx), closeErr)
		assert.True(t, exe.IsClosed())
		require.ErrorIs(t, exe.Close(ctx), closeErr,
			"second Close should report the release error without retrying the plugin Close")
		mockPlugin.AssertExpectations(t)
	})

	t.Run("Close on nil and zero-value executables", func(t *testing.T) {
		var nilExe *Executable
		require.NoError(t, nilExe.Close(t.Context()))

		zeroExe := &Executable{}
		require.NoError(t, zeroExe.Close(t.Context()))
		assert.True(t, zeroExe.IsClosed())

		cancelled, cancel := context.WithCancel(t.Context())
		cancel()
		require.NoError(t, (&Executable{}).Close(cancelled),
			"a zero-value Executable has nothing to release, so a cancelled ctx does not matter")
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
		err := exe.WithPlugin(t.Context(), func(p adapters.CompiledPlugin) error {
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

		err := exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error { return fnErr })
		require.ErrorIs(t, err, fnErr)
	})

	t.Run("nil plugin", func(t *testing.T) {
		exe := &Executable{}
		err := exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error {
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
			useDone <- exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error {
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

	t.Run("pending Close fails new users fast", func(t *testing.T) {
		mockPlugin := new(MockCompiledPlugin)
		mockPlugin.On("Close", mock.Anything).Return(nil).Once()

		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)

		inUse := make(chan struct{})
		release := make(chan struct{})
		useDone := make(chan error, 1)
		go func() {
			useDone <- exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error {
				close(inUse)
				<-release
				return nil
			})
		}()
		<-inUse

		closeDone := make(chan error, 1)
		go func() { closeDone <- exe.Close(t.Context()) }()
		require.Eventually(t, func() bool { return isClosing(exe) }, time.Second, time.Millisecond,
			"Close should mark the executable as closing before waiting")

		// A new user must not queue behind the pending Close.
		newUseDone := make(chan error, 1)
		go func() {
			newUseDone <- exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error {
				t.Error("fn must not run once Close is pending")
				return nil
			})
		}()
		select {
		case err := <-newUseDone:
			require.ErrorIs(t, err, ErrExecutableClosed)
		case <-time.After(time.Second):
			t.Fatal("new WithPlugin blocked behind a pending Close")
		}
		assert.False(t, exe.IsClosed(), "runtime must not be released while still in use")

		close(release)
		require.NoError(t, <-useDone)
		require.NoError(t, <-closeDone)
		assert.True(t, exe.IsClosed())
		mockPlugin.AssertExpectations(t)
	})

	t.Run("concurrent Close calls all wait for the release", func(t *testing.T) {
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
			useDone <- exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error {
				close(inUse)
				<-release
				return nil
			})
		}()
		<-inUse

		firstClose := make(chan error, 1)
		go func() { firstClose <- exe.Close(t.Context()) }()
		require.Eventually(t, func() bool { return isClosing(exe) }, time.Second, time.Millisecond)

		secondClose := make(chan error, 1)
		go func() { secondClose <- exe.Close(t.Context()) }()
		select {
		case <-secondClose:
			t.Fatal("second Close returned before the runtime was released")
		case <-time.After(50 * time.Millisecond):
		}

		close(release)
		require.NoError(t, <-useDone)
		require.NoError(t, <-firstClose)
		require.NoError(t, <-secondClose)
		<-pluginClosed
		mockPlugin.AssertExpectations(t)
	})

	t.Run("many concurrent Close calls release once", func(t *testing.T) {
		// ExecutableContent requires a Close that is safe to repeat and to
		// call concurrently, releasing only once; this pins that for Extism.
		mockPlugin := new(MockCompiledPlugin)
		mockPlugin.On("Close", mock.Anything).Return(nil).Once()
		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)

		const closers = 32
		start := make(chan struct{})
		errs := make(chan error, closers)
		var wg sync.WaitGroup
		for range closers {
			wg.Go(func() {
				<-start
				errs <- exe.Close(t.Context())
			})
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err, "every Close should return the release result")
		}
		assert.True(t, exe.IsClosed())
		mockPlugin.AssertExpectations(t)
	})

	t.Run("concurrent users do not block each other", func(t *testing.T) {
		exe := NewExecutable(wasmBytes, new(MockCompiledPlugin), entryPoint)
		require.NotNil(t, exe)

		firstIn := make(chan struct{})
		secondIn := make(chan struct{})
		firstDone := make(chan error, 1)
		go func() {
			firstDone <- exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error {
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

		err := exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error {
			close(secondIn)
			return nil
		})
		require.NoError(t, err)
		require.NoError(t, <-firstDone)
	})
}

// closeTestTimeout bounds every wait in the Close tests, so a regression
// fails the test instead of hanging the suite.
const closeTestTimeout = 5 * time.Second

// startUser runs a WithPlugin call that blocks until the returned release
// func is called. It returns once fn is running, along with a channel that
// receives WithPlugin's error. The test's cleanup releases the user, so a
// failed assertion cannot leave it blocked.
func startUser(t *testing.T, exe *Executable) (release func(), done <-chan error) {
	t.Helper()
	inUse := make(chan struct{})
	gate := make(chan struct{})
	var once sync.Once
	release = func() { once.Do(func() { close(gate) }) }
	t.Cleanup(release)

	errs := make(chan error, 1)
	go func() {
		errs <- exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error {
			close(inUse)
			<-gate
			return nil
		})
	}()
	select {
	case <-inUse:
	case <-time.After(closeTestTimeout):
		require.FailNow(t, "timed out waiting for WithPlugin to call fn")
	}
	return release, errs
}

// recvWithin returns the next value from ch, failing the test if none
// arrives within closeTestTimeout.
func recvWithin[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(closeTestTimeout):
		require.FailNow(t, "timed out waiting for "+what)
		var zero T
		return zero
	}
}

// isClosing reports whether Close has been called on exe.
func isClosing(exe *Executable) bool {
	return exe.GetExtismByteCode() == nil
}

// liveCtx matches a ctx that is not done, so a release that reuses a
// caller's cancelled ctx fails the mock expectation.
func liveCtx() any {
	return mock.MatchedBy(func(ctx context.Context) bool { return ctx.Err() == nil })
}

func TestExecutable_CloseContext(t *testing.T) {
	t.Parallel()

	wasmBytes := []byte("mock wasm bytes")
	entryPoint := "run"

	t.Run("returns at its deadline and releases after the last user", func(t *testing.T) {
		pluginClosed := make(chan struct{})
		mockPlugin := new(MockCompiledPlugin)
		mockPlugin.On("Close", liveCtx()).Return(nil).Once().
			Run(func(mock.Arguments) { close(pluginClosed) })

		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)
		release, useDone := startUser(t, exe)

		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()
		closeDone := make(chan error, 1)
		go func() { closeDone <- exe.Close(ctx) }()
		require.ErrorIs(t, recvWithin(t, closeDone, "Close to return at its deadline"), context.DeadlineExceeded)
		assert.False(t, exe.IsClosed(), "the plugin must not be released while in use")

		err := exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error {
			t.Error("fn must not run once Close has been called")
			return nil
		})
		require.ErrorIs(t, err, ErrExecutableClosed)

		release()
		require.NoError(t, recvWithin(t, useDone, "the in-flight user to finish"))
		recvWithin(t, pluginClosed, "the plugin to be released after the last user")
		require.Eventually(t, exe.IsClosed, closeTestTimeout, time.Millisecond)
		mockPlugin.AssertExpectations(t)
	})

	t.Run("cancelled ctx with nothing in flight releases synchronously", func(t *testing.T) {
		mockPlugin := new(MockCompiledPlugin)
		mockPlugin.On("Close", liveCtx()).Return(nil).Once()

		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.NoError(t, exe.Close(ctx))
		assert.True(t, exe.IsClosed())
		mockPlugin.AssertExpectations(t)
	})

	t.Run("release error reaches a Close that is still waiting", func(t *testing.T) {
		closeErr := errors.New("close kaboom")
		mockPlugin := new(MockCompiledPlugin)
		mockPlugin.On("Close", liveCtx()).Return(closeErr).Once()

		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)
		release, useDone := startUser(t, exe)

		closeDone := make(chan error, 1)
		go func() { closeDone <- exe.Close(t.Context()) }()
		require.Eventually(t, func() bool { return isClosing(exe) },
			closeTestTimeout, time.Millisecond, "Close should mark the executable as closing")

		release()
		require.NoError(t, recvWithin(t, useDone, "the in-flight user to finish"))
		require.ErrorIs(t, recvWithin(t, closeDone, "Close to return"), closeErr)
		require.ErrorIs(t, exe.Close(t.Context()), closeErr, "a Close after the release returns its error")
		mockPlugin.AssertExpectations(t)
	})

	t.Run("cancelled ctx waits for a release already under way", func(t *testing.T) {
		releasing := make(chan struct{})
		gate := make(chan struct{})
		mockPlugin := new(MockCompiledPlugin)
		mockPlugin.On("Close", liveCtx()).Return(nil).Once().
			Run(func(mock.Arguments) {
				close(releasing)
				<-gate
			})

		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)

		firstClose := make(chan error, 1)
		go func() { firstClose <- exe.Close(context.WithoutCancel(t.Context())) }()
		recvWithin(t, releasing, "the first Close to start the release")

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		secondClose := make(chan error, 1)
		go func() { secondClose <- exe.Close(ctx) }()
		select {
		case err := <-secondClose:
			require.Failf(t, "Close with nothing in flight returned before the release finished", "error: %v", err)
		case <-time.After(20 * time.Millisecond):
		}

		close(gate)
		require.NoError(t, recvWithin(t, secondClose, "the second Close to return"),
			"with nothing in flight, a cancelled ctx must not make Close give up")
		assert.True(t, exe.IsClosed(), "Close should return only once the plugin is released")
		require.NoError(t, recvWithin(t, firstClose, "the first Close to return"))
		mockPlugin.AssertExpectations(t)
	})

	t.Run("release error reaches a later Close after one gave up", func(t *testing.T) {
		closeErr := errors.New("close kaboom")
		mockPlugin := new(MockCompiledPlugin)
		mockPlugin.On("Close", liveCtx()).Return(closeErr).Once()

		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)
		release, useDone := startUser(t, exe)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, exe.Close(ctx), context.Canceled)

		release()
		require.NoError(t, recvWithin(t, useDone, "the in-flight user to finish"),
			"the release error must not leak into the user's result")
		require.Eventually(t, exe.IsClosed, closeTestTimeout, time.Millisecond)
		require.ErrorIs(t, exe.Close(t.Context()), closeErr, "a later Close should report the release error")
		mockPlugin.AssertExpectations(t)
	})

	t.Run("a later Close waits after an earlier one gave up", func(t *testing.T) {
		mockPlugin := new(MockCompiledPlugin)
		mockPlugin.On("Close", liveCtx()).Return(nil).Once()

		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)
		release, useDone := startUser(t, exe)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, exe.Close(ctx), context.Canceled)

		secondClose := make(chan error, 1)
		go func() { secondClose <- exe.Close(t.Context()) }()
		select {
		case err := <-secondClose:
			require.Failf(t, "second Close returned while the plugin was in use", "error: %v", err)
		case <-time.After(20 * time.Millisecond):
		}

		release()
		require.NoError(t, recvWithin(t, useDone, "the in-flight user to finish"))
		require.NoError(t, recvWithin(t, secondClose, "the second Close to return"))
		assert.True(t, exe.IsClosed())
		mockPlugin.AssertExpectations(t)
	})

	// gatedPlugin returns a plugin whose Close signals releasing, then blocks
	// until openGate is called and returns releaseErr.
	gatedPlugin := func(t *testing.T, releaseErr error) (*MockCompiledPlugin, <-chan struct{}, func()) {
		t.Helper()
		releasing := make(chan struct{})
		gate := make(chan struct{})
		openGate := sync.OnceFunc(func() { close(gate) })
		t.Cleanup(openGate)
		mockPlugin := new(MockCompiledPlugin)
		mockPlugin.On("Close", liveCtx()).Return(releaseErr).Once().
			Run(func(mock.Arguments) {
				close(releasing)
				<-gate
			})
		return mockPlugin, releasing, openGate
	}

	t.Run("cancellation is honoured while the last in-flight call releases", func(t *testing.T) {
		releaseErr := errors.New("release kaboom")
		mockPlugin, releasing, openGate := gatedPlugin(t, releaseErr)
		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)
		release, useDone := startUser(t, exe)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		closeDone := make(chan error, 1)
		go func() { closeDone <- exe.Close(ctx) }()
		require.Eventually(t, func() bool { return isClosing(exe) },
			closeTestTimeout, time.Millisecond, "Close should mark the executable as closing")

		release()
		recvWithin(t, releasing, "the last user to start the release")
		cancel()
		require.ErrorIs(t, recvWithin(t, closeDone, "Close to return once its ctx is cancelled"),
			context.Canceled, "the last user is in flight until its release finishes")
		assert.False(t, exe.IsClosed(), "the release is still running")

		openGate()
		require.NoError(t, recvWithin(t, useDone, "the last user to finish its release"))
		assert.True(t, exe.IsClosed())
		require.ErrorIs(t, exe.Close(t.Context()), releaseErr, "a later Close returns the release error")
		mockPlugin.AssertExpectations(t)
	})

	t.Run("a Close arriving during the last call's release honours its ctx", func(t *testing.T) {
		releaseErr := errors.New("release kaboom")
		mockPlugin, releasing, openGate := gatedPlugin(t, releaseErr)
		exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
		require.NotNil(t, exe)
		release, useDone := startUser(t, exe)

		firstClose := make(chan error, 1)
		go func() { firstClose <- exe.Close(t.Context()) }()
		require.Eventually(t, func() bool { return isClosing(exe) },
			closeTestTimeout, time.Millisecond, "Close should mark the executable as closing")
		release()
		recvWithin(t, releasing, "the last user to start the release")

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		secondClose := make(chan error, 1)
		go func() { secondClose <- exe.Close(ctx) }()
		require.ErrorIs(t, recvWithin(t, secondClose, "the second Close to return"),
			context.Canceled, "a Close must not wait past its ctx for the last user's release")

		openGate()
		require.NoError(t, recvWithin(t, useDone, "the last user to finish its release"))
		require.ErrorIs(t, recvWithin(t, firstClose, "the first Close to return"), releaseErr)
		mockPlugin.AssertExpectations(t)
	})
}

func TestExecutable_WatchLifetime(t *testing.T) {
	t.Parallel()

	wasmBytes := []byte("mock wasm bytes")
	entryPoint := "run"

	t.Run("lifetime end closes the executable", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			mockPlugin := new(MockCompiledPlugin)
			mockPlugin.On("Close", liveCtx()).Return(nil).Once()
			exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
			require.NotNil(t, exe)

			done := make(chan struct{})
			exe.WatchLifetime(done)
			synctest.Wait()
			assert.False(t, exe.IsClosed(), "a live lifetime must not close the executable")

			close(done)
			synctest.Wait()
			assert.True(t, exe.IsClosed(), "the end of the lifetime should release the plugin")
			err := exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error {
				t.Error("fn must not run once the lifetime is done")
				return nil
			})
			require.ErrorIs(t, err, ErrExecutableClosed)
			require.NoError(t, exe.Close(t.Context()), "a later Close returns the release result")
			mockPlugin.AssertExpectations(t)
		})
	})

	t.Run("lifetime end waits for in-flight users", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			mockPlugin := new(MockCompiledPlugin)
			mockPlugin.On("Close", liveCtx()).Return(nil).Once()
			exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
			require.NotNil(t, exe)
			done := make(chan struct{})
			exe.WatchLifetime(done)
			release, useDone := startUser(t, exe)

			close(done)
			synctest.Wait()
			assert.False(t, exe.IsClosed(), "the plugin must not be released while in use")

			release()
			require.NoError(t, <-useDone)
			synctest.Wait()
			assert.True(t, exe.IsClosed(), "the last user should release the plugin")
			mockPlugin.AssertExpectations(t)
		})
	})

	t.Run("WithPlugin rejects once the lifetime is done, before the watcher runs", func(t *testing.T) {
		// No watcher is started here, so only the lifetimeDone check can
		// reject the call.
		mockPlugin := new(MockCompiledPlugin)
		done := make(chan struct{})
		exe := &Executable{byteCode: mockPlugin, lifetimeDone: done}

		close(done)
		err := exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error {
			t.Error("fn must not run once the lifetime is done")
			return nil
		})
		require.ErrorIs(t, err, ErrExecutableClosed)
		mockPlugin.AssertExpectations(t)
	})

	t.Run("explicit Close stops the watcher", func(t *testing.T) {
		// synctest.Test fails if the watcher goroutine is still blocked when
		// the test returns, so this also checks that it does not leak.
		synctest.Test(t, func(t *testing.T) {
			mockPlugin := new(MockCompiledPlugin)
			mockPlugin.On("Close", liveCtx()).Return(nil).Once()
			exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
			require.NotNil(t, exe)

			exe.WatchLifetime(make(chan struct{}))
			require.NoError(t, exe.Close(t.Context()))
			synctest.Wait()
			mockPlugin.AssertExpectations(t)
		})
	})

	t.Run("second WatchLifetime is a no-op", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			mockPlugin := new(MockCompiledPlugin)
			mockPlugin.On("Close", liveCtx()).Return(nil).Once()
			exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
			require.NotNil(t, exe)

			first := make(chan struct{})
			second := make(chan struct{})
			exe.WatchLifetime(first)
			exe.WatchLifetime(second)
			close(second)
			synctest.Wait()
			assert.False(t, exe.IsClosed(), "only the first lifetime is watched")
			require.NoError(t, exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error { return nil }))

			close(first)
			synctest.Wait()
			assert.True(t, exe.IsClosed())
			mockPlugin.AssertExpectations(t)
		})
	})

	t.Run("nil done is a no-op", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			mockPlugin := new(MockCompiledPlugin)
			mockPlugin.On("Close", liveCtx()).Return(nil).Once()
			exe := NewExecutable(wasmBytes, mockPlugin, entryPoint)
			require.NotNil(t, exe)

			exe.WatchLifetime(nil)
			require.NoError(t, exe.WithPlugin(t.Context(), func(adapters.CompiledPlugin) error { return nil }))

			done := make(chan struct{})
			exe.WatchLifetime(done)
			close(done)
			synctest.Wait()
			assert.True(t, exe.IsClosed(), "a nil done must not use up the one watcher")
			mockPlugin.AssertExpectations(t)
		})
	})

	t.Run("zero-value executable", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			exe := &Executable{}
			done := make(chan struct{})
			exe.WatchLifetime(done)
			close(done)
			synctest.Wait()
			assert.True(t, exe.IsClosed())
		})
	})
}
