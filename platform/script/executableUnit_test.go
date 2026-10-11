package script

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	engineTypes "github.com/robbyt/go-polyscript/engines/types"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var emptyScriptData = make(map[string]any)

// Mock implementations
type mockLoader struct {
	mock.Mock
}

func (m *mockLoader) GetReader(ctx context.Context) (io.ReadCloser, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *mockLoader) GetSourceURL() *url.URL {
	args := m.Called()
	return args.Get(0).(*url.URL)
}

type mockReadCloser struct {
	mock.Mock
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func (m *mockReadCloser) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestVersionMethods(t *testing.T) {
	t.Run("EngineType", func(t *testing.T) {
		mockContent := new(MockExecutableContent)
		expectedType := engineTypes.Risor
		mockContent.On("EngineType").Return(expectedType)

		exe := &ExecutableUnit{
			content: mockContent,
		}

		engineType := exe.EngineType()
		require.Equal(t, expectedType, engineType, "Expected engine type to match")
		mockContent.AssertExpectations(t)
	})

	t.Run("GetCompiler", func(t *testing.T) {
		mockCompiler := new(MockCompiler)
		exe := &ExecutableUnit{
			compiler: mockCompiler,
		}

		compiler := exe.GetCompiler()
		require.Equal(t, mockCompiler, compiler, "Expected compiler to match")
	})

	t.Run("GetContent", func(t *testing.T) {
		mockContent := new(MockExecutableContent)

		exe := &ExecutableUnit{
			content: mockContent,
		}

		content := exe.GetContent()
		require.Equal(t, mockContent, content, "Expected content to match the mock content")
		mockContent.AssertExpectations(t)
	})

	t.Run("GetCreatedAt", func(t *testing.T) {
		createdAt := time.Now()
		exe := &ExecutableUnit{
			createdAt: createdAt,
		}

		timestamp := exe.GetCreatedAt()
		require.Equal(t, createdAt, timestamp, "Expected CreatedAt to match the provided timestamp")
	})
}

func TestNewVersion(t *testing.T) {
	t.Parallel()
	// Create test logger
	logHandler := slog.NewTextHandler(os.Stdout, nil)
	t.Run("ValidID", func(t *testing.T) {
		// Create script content and reader
		scriptContent := "print('Hello, World!')"

		lod, err := loader.NewFromString(scriptContent)
		require.NoError(t, err, "Expected no error when creating loader")

		reader, err := lod.GetReader(t.Context())
		require.NoError(t, err, "Expected no error when getting reader")

		// Create mock loader instead of real loader
		mockLoader := new(mockLoader)
		mockLoader.On("GetReader", mock.Anything).Return(reader, nil)

		// Setup mock compiler with same reader instance
		comp := new(MockCompiler)
		comp.On("Compile", mock.Anything, reader).Return(&MockExecutableContent{}, nil)

		// Create executable unit
		exe, err := NewExecutableUnit(
			t.Context(),
			logHandler,
			t.Name(),
			mockLoader,
			comp,
			data.NewStaticProvider(emptyScriptData),
		)
		require.NoError(t, err, "Expected no error when creating executable unit")
		require.NotNil(t, exe, "Expected executable unit to be non-nil")
		require.Equal(t, t.Name(), exe.GetID(), "Expected ID to match")

		// Verify all mocks
		mockLoader.AssertExpectations(t)
		comp.AssertExpectations(t)
	})

	t.Run("ValidContent", func(t *testing.T) {
		scriptBody := "valid script content"
		lod, err := loader.NewFromString(scriptBody)
		require.NoError(t, err, "Expected no error when creating a new loader with valid content")

		reader, err := lod.GetReader(t.Context())
		require.NoError(t, err, "Expected no error when getting reader")

		comp := new(MockCompiler)
		mockContent := new(MockExecutableContent)
		comp.On("Compile", mock.Anything, reader).Return(mockContent, nil).Once()

		exe, err := NewExecutableUnit(
			t.Context(),
			logHandler,
			t.Name(),
			lod,
			comp,
			data.NewStaticProvider(emptyScriptData),
		)
		require.NoError(t, err, "Expected no error when creating a new version with valid content")
		require.NotNil(t, exe, "Expected version to be non-nil")
		require.Equal(
			t,
			mockContent,
			exe.GetContent(),
			"Expected content to match the mock content",
		)
		require.NotNil(t, exe.GetLoader().GetSourceURL(), "Expected SourceURI to be non-nil")
		require.Contains(t, exe.GetLoader().GetSourceURL().String(), "string://inline/")
		require.WithinDuration(
			t,
			time.Now(),
			exe.GetCreatedAt(),
			time.Second,
			"Expected CreatedAt to be within the last second",
		)

		comp.AssertExpectations(t)
		mockContent.AssertExpectations(t)
	})

	t.Run("ValidationError", func(t *testing.T) {
		scriptBody := "invalid script content"
		lod, err := loader.NewFromString(scriptBody)
		require.NoError(t, err, "Expected no error when creating a new loader with empty content")

		reader, err := lod.GetReader(t.Context())
		require.NoError(t, err, "Expected no error when getting reader")

		comp := new(MockCompiler)
		validationError := errors.New("validation failed")
		comp.On("Compile", mock.Anything, reader).Return(nil, validationError).Once()

		exe, err := NewExecutableUnit(
			t.Context(),
			logHandler,
			t.Name(),
			lod,
			comp,
			data.NewStaticProvider(emptyScriptData),
		)
		require.Error(t, err)
		require.Nil(t, exe)
		require.ErrorIs(t, err, validationError)

		comp.AssertExpectations(t)
	})

	t.Run("EmptyVersionID_ReturnsChecksum", func(t *testing.T) {
		// Create script content
		scriptContent := "test content"

		// Create loader and get reader
		lod, err := loader.NewFromString(scriptContent)
		require.NoError(t, err, "Expected no error when creating loader")

		reader, err := lod.GetReader(t.Context())
		require.NoError(t, err, "Expected no error when getting reader")

		// Create mock loader instead of real loader
		mockLoader := new(mockLoader)
		mockLoader.On("GetReader", mock.Anything).Return(reader, nil)

		// Setup mock compiler with same reader instance
		mockCompiler := new(MockCompiler)
		mockContent := new(MockExecutableContent)
		// Add expectation for GetSource
		mockContent.On("GetSource").Return(scriptContent)
		mockCompiler.On("Compile", mock.Anything, reader).Return(mockContent, nil)

		// Create executable unit with empty ID
		exe, err := NewExecutableUnit(
			t.Context(),
			logHandler,
			"",
			mockLoader,
			mockCompiler,
			data.NewStaticProvider(emptyScriptData),
		)
		require.NoError(t, err)
		require.NotNil(t, exe)

		// Verify version ID is set from content checksum
		versionID := exe.GetID()
		require.NotEmpty(t, versionID, "Expected version ID to be non-empty")
		require.Len(
			t,
			versionID, checksumLength,
			"Expected version ID length to match checksum length",
		)

		// Verify all mocks
		mockLoader.AssertExpectations(t)
		mockCompiler.AssertExpectations(t)
		mockContent.AssertExpectations(t)
	})

	t.Run("NilCompiler", func(t *testing.T) {
		exe, err := NewExecutableUnit(
			t.Context(),
			logHandler,
			"test",
			&mockLoader{},
			nil,
			data.NewStaticProvider(emptyScriptData),
		)
		require.Error(t, err)
		require.Nil(t, exe)
		require.Contains(t, err.Error(), "compiler is nil")
	})

	t.Run("EmptyContent", func(t *testing.T) {
		loader, err := loader.NewFromString("")

		require.Nil(t, loader)
		require.Error(t, err)
	})

	t.Run("EmptyContentFromReader", func(t *testing.T) {
		// Setup mock reader with proper expectations
		mockReader := new(mockReadCloser)

		// Setup mock loader with source URL
		mockLoader := new(mockLoader)
		mockLoader.On("GetReader", mock.Anything).Return(mockReader, nil)

		// Setup mock compiler with expected error
		mockCompiler := new(MockCompiler)
		mockCompiler.On("Compile", mock.Anything, mockReader).Return(nil, errors.New("empty content"))
		// Create executable unit
		exe, err := NewExecutableUnit(
			t.Context(),
			logHandler,
			"test",
			mockLoader,
			mockCompiler,
			data.NewStaticProvider(emptyScriptData),
		)
		require.Error(t, err)
		require.Nil(t, exe)

		// Verify all mocks
		mockReader.AssertExpectations(t)
		mockLoader.AssertExpectations(t)
		mockCompiler.AssertExpectations(t)
	})

	t.Run("GetReaderError", func(t *testing.T) {
		// Real loaders follow the Go convention: nil reader on error.
		// Modeling it that way here keeps the mock contract honest.
		mockLoader := new(mockLoader)
		mockLoader.On("GetReader", mock.Anything).Return(nil, errors.New("get reader error")).Once()

		exe, err := NewExecutableUnit(
			t.Context(),
			logHandler,
			"test",
			mockLoader,
			new(MockCompiler),
			data.NewStaticProvider(emptyScriptData),
		)
		require.Error(t, err)
		require.Nil(t, exe)
		mockLoader.AssertExpectations(t)
	})

	t.Run("ReaderError", func(t *testing.T) {
		// Setup mock reader with read error
		mockReader := new(mockReadCloser)

		// Setup mock loader
		mockLoader := new(mockLoader)
		mockLoader.On("GetReader", mock.Anything).Return(mockReader, nil).Once()

		// Setup mock compiler with same reader instance
		mockCompiler := new(MockCompiler)
		mockCompiler.On("Compile", mock.Anything, mockReader).Return(nil, errors.New("compile failed")).Once()

		// Create executable unit
		exe, err := NewExecutableUnit(
			t.Context(),
			logHandler,
			"test",
			mockLoader,
			mockCompiler,
			data.NewStaticProvider(emptyScriptData),
		)
		require.Error(t, err)
		require.Nil(t, exe)

		// Verify all mocks
		mockReader.AssertExpectations(t)
		mockLoader.AssertExpectations(t)
		mockCompiler.AssertExpectations(t)
	})
}

func TestExecutableUnit_String(t *testing.T) {
	t.Parallel()

	t.Run("String method", func(t *testing.T) {
		mockLoader := new(mockLoader)
		mockCompiler := new(MockCompiler)
		mockContent := new(MockExecutableContent)

		exe := &ExecutableUnit{
			id:           "testID",
			createdAt:    time.Now(),
			scriptLoader: mockLoader,
			content:      mockContent,
			compiler:     mockCompiler,
		}

		expectedString := fmt.Sprintf(
			"ExecutableUnit{ID: %s, CreatedAt: %s, Compiler: %s, Loader: %s}",
			exe.id,
			exe.createdAt,
			exe.compiler,
			exe.scriptLoader,
		)

		require.Equal(t, expectedString, exe.String(), "Expected string representation to match")
	})
}

func TestNewVersionWithScriptData(t *testing.T) {
	t.Parallel()
	// Create test logger
	logHandler := slog.NewTextHandler(os.Stdout, nil)

	t.Run("ValidScriptData", func(t *testing.T) {
		// Script content
		scriptContent := "print('Hello, World!')"

		// Create mock compiler and mock content
		mockCompiler := new(MockCompiler)
		mockContent := new(MockExecutableContent)

		// Create lod and set mock expectations
		lod, err := loader.NewFromString(scriptContent)
		require.NoError(t, err, "Expected no error when creating loader")

		reader, err := lod.GetReader(t.Context())
		require.NoError(t, err, "Expected no error when getting reader")

		mockCompiler.On("Compile", mock.Anything, reader).Return(mockContent, nil).Once()

		// Create loader directly from string instead of file
		loader, err := loader.NewFromString(scriptContent)
		require.NoError(t, err, "Expected no error when creating loader")

		// Test data
		scriptData := map[string]any{
			"key1": "value1",
			"key2": 42,
		}

		// Create executable unit
		exe, err := NewExecutableUnit(
			t.Context(),
			logHandler,
			t.Name(),
			loader,
			mockCompiler,
			data.NewStaticProvider(scriptData),
		)
		require.NoError(t, err, "Expected no error creating executable unit")
		require.NotNil(t, exe, "Expected executable unit to be non-nil")

		dataFromProvider, err := exe.dataProvider.GetData(t.Context())
		require.NoError(t, err, "Expected no error when getting data from provider")
		require.Equal(t, scriptData, dataFromProvider, "Expected script data to match")

		// Verify all mocks
		mockCompiler.AssertExpectations(t)
		mockContent.AssertExpectations(t)
	})

	t.Run("EmptyScriptData", func(t *testing.T) {
		scriptBody := "valid script content"

		lod, err := loader.NewFromString(scriptBody)
		require.NoError(t, err, "Expected no error when creating a new loader with valid content")

		reader, err := lod.GetReader(t.Context())
		require.NoError(t, err, "Expected no error when getting reader")

		comp := new(MockCompiler)
		mockContent := new(MockExecutableContent)

		comp.On("Compile", mock.Anything, reader).Return(mockContent, nil).Once()

		exe, err := NewExecutableUnit(
			t.Context(),
			logHandler,
			t.Name(),
			lod,
			comp,
			data.NewStaticProvider(nil),
		)
		require.NoError(t, err,
			"Expected no error when creating a new version with nil script data",
		)

		require.NotNil(t, exe, "Expected version to be non-nil")
		dataFromProvider, err := exe.dataProvider.GetData(t.Context())
		require.NoError(t, err, "Expected no error when getting data from provider")
		require.Equal(
			t,
			emptyScriptData,
			dataFromProvider,
			"Expected script data to match empty map",
		)

		comp.AssertExpectations(t)
		mockContent.AssertExpectations(t)
	})
}

func TestNewExecutableUnit_NilHandler(t *testing.T) {
	t.Parallel()

	lod, err := loader.NewFromString("any content")
	require.NoError(t, err)

	reader, err := lod.GetReader(t.Context())
	require.NoError(t, err)

	mockLdr := new(mockLoader)
	mockLdr.On("GetReader", mock.Anything).Return(reader, nil)

	comp := new(MockCompiler)
	comp.On("Compile", mock.Anything, reader).Return(&MockExecutableContent{}, nil)

	exe, err := NewExecutableUnit(t.Context(), nil, t.Name(), mockLdr, comp, data.NewStaticProvider(emptyScriptData))
	require.NoError(t, err)
	require.NotNil(t, exe)
	require.Equal(t, t.Name(), exe.GetID())

	mockLdr.AssertExpectations(t)
	comp.AssertExpectations(t)
}

func TestExecutableUnit_Close(t *testing.T) {
	t.Parallel()

	t.Run("closes content that implements Close", func(t *testing.T) {
		ctx := t.Context()
		content := new(MockClosableExecutableContent)
		content.On("Close", ctx).Return(nil).Once()

		exe := &ExecutableUnit{content: content}
		require.NoError(t, exe.Close(ctx))
		content.AssertExpectations(t)
	})

	t.Run("propagates content Close error", func(t *testing.T) {
		ctx := t.Context()
		closeErr := errors.New("close kaboom")
		content := new(MockClosableExecutableContent)
		content.On("Close", ctx).Return(closeErr).Once()

		exe := &ExecutableUnit{content: content}
		require.ErrorIs(t, exe.Close(ctx), closeErr)
		content.AssertExpectations(t)
	})

	t.Run("repeated Close closes the content once", func(t *testing.T) {
		closeErr := errors.New("close kaboom")
		content := &countingClosableContent{err: closeErr}
		exe := &ExecutableUnit{content: content}

		require.ErrorIs(t, exe.Close(t.Context()), closeErr)
		require.ErrorIs(t, exe.Close(t.Context()), closeErr, "a later Close should return the first close's result")
		assert.Equal(t, int32(1), content.closeCalls.Load(), "the content should be closed only once")
	})

	t.Run("concurrent Close calls close the content once", func(t *testing.T) {
		gate := make(chan struct{})
		content := &blockingClosableContent{gate: gate}
		exe := &ExecutableUnit{content: content}

		const closers = 16
		errs := make(chan error, closers)
		for range closers {
			go func() { errs <- exe.Close(t.Context()) }()
		}
		// Every Close blocks behind the first until its content close returns.
		select {
		case err := <-errs:
			t.Fatalf("a Close returned before the content close finished: %v", err)
		case <-time.After(50 * time.Millisecond):
		}
		close(gate)
		for range closers {
			select {
			case err := <-errs:
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for Close to return")
			}
		}
		assert.Equal(t, int32(1), content.closeCalls.Load(), "the content should be closed only once")
	})

	t.Run("Close racing the lifetime hook closes the content once", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			content := &countingClosableContent{}
			exe := newUnitWithContent(t, nil, content)
			lifetime, cancel := context.WithCancel(t.Context())
			require.NoError(t, exe.BindLifetime(lifetime))

			cancel()
			require.NoError(t, exe.Close(t.Context()))
			synctest.Wait()
			assert.Equal(t, int32(1), content.closeCalls.Load(), "the hook and Close should close the content once between them")
			runtime.KeepAlive(exe) // keep the unit reachable so its GC cleanup cannot run
		})
	})

	t.Run("no-op for content without Close", func(t *testing.T) {
		content := new(MockExecutableContent)
		exe := &ExecutableUnit{content: content}
		require.NoError(t, exe.Close(t.Context()))
		content.AssertExpectations(t)
	})

	t.Run("nil content", func(t *testing.T) {
		exe := &ExecutableUnit{}
		require.NoError(t, exe.Close(t.Context()))
	})

	t.Run("nil unit", func(t *testing.T) {
		var exe *ExecutableUnit
		require.NoError(t, exe.Close(t.Context()))
	})
}

// syncBuffer is a bytes.Buffer safe for a log handler that writes from
// another goroutine, such as a context.AfterFunc callback.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// newUnitWithContent builds an ExecutableUnit around content through
// NewExecutableUnit, logging to handler.
func newUnitWithContent(t *testing.T, handler slog.Handler, content ExecutableContent) *ExecutableUnit {
	t.Helper()
	ldr := new(mockLoader)
	ldr.On("GetReader", mock.Anything).Return(io.NopCloser(strings.NewReader("script")), nil)
	comp := new(MockCompiler)
	comp.On("Compile", mock.Anything, mock.Anything).Return(content, nil)

	exe, err := NewExecutableUnit(
		t.Context(), handler, t.Name(), ldr, comp, data.NewStaticProvider(emptyScriptData),
	)
	require.NoError(t, err)
	ldr.AssertExpectations(t)
	comp.AssertExpectations(t)
	return exe
}

// bindUnusedUnit builds a unit around content, binds it to a new live
// lifetime and drops it, so the only remaining reference to the unit is the
// lifetime hook. It returns the func that ends the lifetime.
func bindUnusedUnit(t *testing.T, content ExecutableContent) context.CancelFunc {
	t.Helper()
	exe := newUnitWithContent(t, nil, content)
	lifetime, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	require.NoError(t, exe.BindLifetime(lifetime))
	return cancel
}

func TestExecutableUnit_BindLifetime(t *testing.T) {
	t.Parallel()

	t.Run("lifetime end closes the content once", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			content := &countingClosableContent{}
			exe := newUnitWithContent(t, nil, content)
			lifetime, cancel := context.WithCancel(t.Context())
			defer cancel()

			require.NoError(t, exe.BindLifetime(lifetime))
			synctest.Wait()
			assert.Equal(t, int32(0), content.closeCalls.Load(), "content must stay open while the lifetime is live")

			cancel()
			synctest.Wait()
			assert.Equal(t, int32(1), content.closeCalls.Load(), "lifetime end should close the content")

			cancel()
			synctest.Wait()
			assert.Equal(t, int32(1), content.closeCalls.Load(), "the hook should fire only once")
			runtime.KeepAlive(exe) // keep the unit reachable so its GC cleanup cannot close the content first
		})
	})

	t.Run("Close stops the lifetime hook", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			content := &countingClosableContent{}
			exe := newUnitWithContent(t, nil, content)
			lifetime, cancel := context.WithCancel(t.Context())
			defer cancel()

			require.NoError(t, exe.BindLifetime(lifetime))
			require.NoError(t, exe.Close(t.Context()))
			assert.Equal(t, int32(1), content.closeCalls.Load(), "Close should close the content")

			cancel()
			synctest.Wait()
			assert.Equal(t, int32(1), content.closeCalls.Load(),
				"lifetime end after Close must not close the content again")
		})
	})

	t.Run("hook logs a content close error", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var logs syncBuffer
			content := &countingClosableContent{err: errors.New("close kaboom")}
			exe := newUnitWithContent(t, slog.NewTextHandler(&logs, nil), content)
			lifetime, cancel := context.WithCancel(t.Context())
			defer cancel()

			require.NoError(t, exe.BindLifetime(lifetime))
			cancel()
			synctest.Wait()
			assert.Equal(t, int32(1), content.closeCalls.Load())
			assert.Contains(t, logs.String(), "close kaboom", "a close error from the hook should be logged")
			runtime.KeepAlive(exe)
		})
	})

	t.Run("content without Close ignores the lifetime", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			content := new(MockExecutableContent)
			exe := newUnitWithContent(t, nil, content)
			lifetime, cancel := context.WithCancel(t.Context())

			require.NoError(t, exe.BindLifetime(lifetime))
			cancel()
			synctest.Wait()
			require.NoError(t, exe.Close(t.Context()))
			content.AssertExpectations(t)
		})
	})

	t.Run("rebinding replaces the earlier lifetime", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			content := &countingClosableContent{}
			exe := newUnitWithContent(t, nil, content)
			first, cancelFirst := context.WithCancel(t.Context())
			defer cancelFirst()
			second, cancelSecond := context.WithCancel(t.Context())
			defer cancelSecond()

			require.NoError(t, exe.BindLifetime(first))
			require.NoError(t, exe.BindLifetime(second))
			cancelFirst()
			synctest.Wait()
			assert.Equal(t, int32(0), content.closeCalls.Load(), "the replaced lifetime must not close the content")

			cancelSecond()
			synctest.Wait()
			assert.Equal(t, int32(1), content.closeCalls.Load())
			runtime.KeepAlive(exe)
		})
	})

	t.Run("lifetime does not keep an unused unit in memory", func(t *testing.T) {
		content := &countingClosableContent{}
		collected := make(chan struct{})
		runtime.AddCleanup(content, func(ch chan struct{}) { close(ch) }, collected)
		bindUnusedUnit(t, content)

		// The unit's cleanup closes the content and lets it go; only then can
		// the content itself be collected.
		deadline := time.After(5 * time.Second)
		for {
			runtime.GC()
			select {
			case <-collected:
				return
			case <-deadline:
				t.Fatal("a unit bound to a live lifetime but no longer used should be garbage collected")
			case <-time.After(10 * time.Millisecond):
			}
		}
	})

	t.Run("unused unit has its content closed once collected", func(t *testing.T) {
		content := &countingClosableContent{}
		cancel := bindUnusedUnit(t, content)
		require.Eventually(t, func() bool {
			runtime.GC()
			return content.closeCalls.Load() == 1
		}, 5*time.Second, 10*time.Millisecond, "collecting a bound, unclosed unit should close its content")

		cancel()
		time.Sleep(20 * time.Millisecond) // let a hook that wrongly survived run
		assert.Equal(t, int32(1), content.closeCalls.Load(), "the lifetime hook must be gone once the unit is collected")
	})

	t.Run("unit dropped as its lifetime ends still has its content closed", func(t *testing.T) {
		content := &countingClosableContent{}
		cancel := bindUnusedUnit(t, content)
		// End the lifetime and collect the unit at once, so the hook and the
		// unit's GC cleanup race; whichever wins, the content is closed once.
		// This is a smoke test: the window where the unit is collected after
		// the hook starts is too narrow to hit on demand. The guarantee itself
		// is structural: the hook refers to the content, never to the unit.
		cancel()
		require.Eventually(t, func() bool {
			runtime.GC()
			return content.closeCalls.Load() == 1
		}, 5*time.Second, 10*time.Millisecond, "the content of a unit dropped as its lifetime ends should be closed")
		for range 5 {
			runtime.GC()
			time.Sleep(10 * time.Millisecond)
		}
		assert.Equal(t, int32(1), content.closeCalls.Load(), "the content should be closed exactly once")
	})

	t.Run("collecting a closed unit does not close its content again", func(t *testing.T) {
		content := &countingClosableContent{}
		func() {
			exe := newUnitWithContent(t, nil, content)
			lifetime, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			require.NoError(t, exe.BindLifetime(lifetime))
			require.NoError(t, exe.Close(lifetime))
		}()
		for range 5 {
			runtime.GC()
			time.Sleep(10 * time.Millisecond)
		}
		assert.Equal(t, int32(1), content.closeCalls.Load(), "only the explicit Close should close the content")
	})

	t.Run("failed rebind drops the earlier lifetime", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			content := &countingClosableContent{}
			exe := newUnitWithContent(t, nil, content)
			first, cancelFirst := context.WithCancel(t.Context())
			defer cancelFirst()
			require.NoError(t, exe.BindLifetime(first))

			ended, cancelEnded := context.WithCancel(t.Context())
			cancelEnded()
			require.ErrorIs(t, exe.BindLifetime(ended), context.Canceled)
			assert.Equal(t, int32(1), content.closeCalls.Load(), "a bind to an ended lifetime should close the content")

			cancelFirst()
			synctest.Wait()
			assert.Equal(t, int32(1), content.closeCalls.Load(), "the earlier lifetime's hook should be gone")
			runtime.KeepAlive(exe) // keep the unit reachable so its GC cleanup cannot close the content first
		})
	})

	t.Run("nil lifetime", func(t *testing.T) {
		content := &countingClosableContent{}
		exe := newUnitWithContent(t, nil, content)
		var lifetime context.Context
		require.Error(t, exe.BindLifetime(lifetime))
		assert.Equal(t, int32(0), content.closeCalls.Load())
	})

	t.Run("nil unit", func(t *testing.T) {
		var exe *ExecutableUnit
		require.NoError(t, exe.BindLifetime(t.Context()))
	})
}

// endsWhenWatched is a lifetime that is still live when first checked but
// ends as soon as something watches its Done channel, as context.AfterFunc
// does while registering a hook. It models a lifetime that ends during
// BindLifetime, between its check and the hook's registration.
type endsWhenWatched struct {
	context.Context
	cancel context.CancelFunc
}

func (c endsWhenWatched) Done() <-chan struct{} {
	c.cancel()
	return c.Context.Done()
}

// endsAfterRegistering is a lifetime that is live at BindLifetime's first
// check and done at its second, and whose AfterFunc never runs the hook. So
// BindLifetime's stop always unregisters the hook, and BindLifetime must
// close the content itself.
type endsAfterRegistering struct {
	errCalls *atomic.Int32
}

func (endsAfterRegistering) Deadline() (time.Time, bool) { return time.Time{}, false }
func (endsAfterRegistering) Done() <-chan struct{}       { return make(chan struct{}) }
func (endsAfterRegistering) Value(any) any               { return nil }

func (c endsAfterRegistering) Err() error {
	if c.errCalls.Add(1) == 1 {
		return nil
	}
	return context.Canceled
}

// AfterFunc makes context.AfterFunc register through this lifetime, which
// never calls f, so the hook can only be unregistered.
func (endsAfterRegistering) AfterFunc(func()) func() bool {
	return func() bool { return true }
}

func TestExecutableUnit_BindLifetime_AlreadyDone(t *testing.T) {
	t.Parallel()

	errShutdown := errors.New("shutting down")
	closeErr := errors.New("close kaboom")

	tests := []struct {
		name     string
		lifetime func() context.Context
		closeErr error
		wantErrs []error
	}{
		{
			name: "cancelled",
			lifetime: func() context.Context {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				return ctx
			},
			wantErrs: []error{context.Canceled},
		},
		{
			name: "cancelled with a cause",
			lifetime: func() context.Context {
				ctx, cancel := context.WithCancelCause(t.Context())
				cancel(errShutdown)
				return ctx
			},
			wantErrs: []error{context.Canceled, errShutdown},
		},
		{
			name: "deadline exceeded",
			lifetime: func() context.Context {
				ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
				cancel()
				return ctx
			},
			wantErrs: []error{context.DeadlineExceeded},
		},
		{
			name: "ends while the hook is registered",
			lifetime: func() context.Context {
				ctx, cancel := context.WithCancel(t.Context())
				t.Cleanup(cancel)
				return endsWhenWatched{Context: ctx, cancel: cancel}
			},
			wantErrs: []error{context.Canceled},
		},
		{
			name: "ends after the hook is registered, hook never run",
			lifetime: func() context.Context {
				return endsAfterRegistering{errCalls: new(atomic.Int32)}
			},
			wantErrs: []error{context.Canceled},
		},
		{
			name: "content close error is joined",
			lifetime: func() context.Context {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				return ctx
			},
			closeErr: closeErr,
			wantErrs: []error{context.Canceled, closeErr},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content := &countingClosableContent{err: tc.closeErr}
			exe := newUnitWithContent(t, nil, content)

			err := exe.BindLifetime(tc.lifetime())
			require.Error(t, err)
			for _, want := range tc.wantErrs {
				require.ErrorIs(t, err, want)
			}
			assert.Equal(t, int32(1), content.closeCalls.Load(),
				"an already-done lifetime should close the content before returning")
		})
	}
}
