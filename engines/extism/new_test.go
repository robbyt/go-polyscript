package extism

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"testing"

	"github.com/robbyt/go-polyscript/engines/extism/compiler"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newWASMLoader(t *testing.T) *loaderMock {
	t.Helper()
	mockLoader := new(loaderMock)
	mockURL, err := url.Parse("file:///test-wasm-file.wasm")
	require.NoError(t, err)
	mockLoader.On("GetSourceURL").Return(mockURL)
	wasmBytes := wasmdata.TestModule
	mockLoader.On("GetReader", mock.Anything).Return(io.NopCloser(bytes.NewReader(wasmBytes)), nil)
	return mockLoader
}

func newErrorLoader(t *testing.T, msg string) *loaderMock {
	t.Helper()
	mockLoader := new(loaderMock)
	mockURL, err := url.Parse("file:///test-wasm-file.wasm")
	require.NoError(t, err)
	mockLoader.On("GetSourceURL").Return(mockURL)
	mockLoader.On("GetReader", mock.Anything).Return(nil, errors.New(msg))
	return mockLoader
}

func TestFromExtismLoader_RequiresEntryPoint(t *testing.T) {
	mockLoader := new(loaderMock)
	eval, err := FromExtismLoader(t.Context(), mockLoader)
	require.Error(t, err)
	require.Nil(t, eval)
	require.ErrorIs(t, err, ErrEntryPointRequired)
	// Loader must not be touched when entry-point validation fails.
	mockLoader.AssertNotCalled(t, "GetReader")
	mockLoader.AssertNotCalled(t, "GetSourceURL")
}

func TestFromExtismLoader_EmptyEntryPointStillRejected(t *testing.T) {
	mockLoader := new(loaderMock)
	eval, err := FromExtismLoader(t.Context(), mockLoader, WithEntryPoint(""))
	require.Error(t, err)
	require.Nil(t, eval)
	require.ErrorIs(t, err, ErrEntryPointRequired)
}

func TestFromExtismLoader_NoOptionsBeyondEntryPoint(t *testing.T) {
	mockLoader := newWASMLoader(t)
	eval, err := FromExtismLoader(t.Context(), mockLoader, WithEntryPoint(wasmdata.EntrypointGreet))
	require.NoError(t, err)
	require.NotNil(t, eval)
	assert.Equal(t, "extism.Evaluator", eval.String())
	mockLoader.AssertExpectations(t)
}

func TestFromExtismLoader_WithLogHandler(t *testing.T) {
	mockLoader := newWASMLoader(t)
	handler := slog.NewTextHandler(os.Stdout, nil)
	eval, err := FromExtismLoader(
		t.Context(), mockLoader,
		WithEntryPoint(wasmdata.EntrypointGreet),
		WithLogHandler(handler),
	)
	require.NoError(t, err)
	require.NotNil(t, eval)
	mockLoader.AssertExpectations(t)
}

func TestFromExtismLoader_NilLogHandler(t *testing.T) {
	mockLoader := newWASMLoader(t)
	eval, err := FromExtismLoader(
		t.Context(), mockLoader,
		WithEntryPoint(wasmdata.EntrypointGreet),
		WithLogHandler(nil),
	)
	require.NoError(t, err)
	require.NotNil(t, eval)
	mockLoader.AssertExpectations(t)
}

func TestFromExtismLoader_WithStaticData(t *testing.T) {
	mockLoader := newWASMLoader(t)
	eval, err := FromExtismLoader(
		t.Context(), mockLoader,
		WithEntryPoint(wasmdata.EntrypointGreet),
		WithStaticData(map[string]any{"input": "World"}),
	)
	require.NoError(t, err)
	require.NotNil(t, eval)
	mockLoader.AssertExpectations(t)
}

func TestFromExtismLoader_WithDataProvider(t *testing.T) {
	mockLoader := newWASMLoader(t)
	provider := data.NewContextProvider("test_key")
	eval, err := FromExtismLoader(
		t.Context(), mockLoader,
		WithEntryPoint(wasmdata.EntrypointGreet),
		WithDataProvider(provider),
	)
	require.NoError(t, err)
	require.NotNil(t, eval)
	mockLoader.AssertExpectations(t)
}

func TestFromExtismLoader_DataProviderBeatsStaticData(t *testing.T) {
	mockLoader := newWASMLoader(t)
	provider := data.NewContextProvider("sentinel")
	eval, err := FromExtismLoader(
		t.Context(), mockLoader,
		WithEntryPoint(wasmdata.EntrypointGreet),
		WithStaticData(map[string]any{"ignored": true}),
		WithDataProvider(provider),
	)
	require.NoError(t, err)
	require.NotNil(t, eval)
	mockLoader.AssertExpectations(t)
}

func TestFromExtismLoader_NilOption(t *testing.T) {
	mockLoader := newWASMLoader(t)
	var nilOpt Option
	eval, err := FromExtismLoader(
		t.Context(), mockLoader,
		nilOpt,
		WithEntryPoint(wasmdata.EntrypointGreet),
	)
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromExtismLoader_LoaderError(t *testing.T) {
	mockLoader := newErrorLoader(t, "failed to load WASM")
	eval, err := FromExtismLoader(t.Context(), mockLoader, WithEntryPoint(wasmdata.EntrypointGreet))
	require.Error(t, err)
	require.Nil(t, eval)
	mockLoader.AssertExpectations(t)
}

func TestFromExtismLoader_NilSourceURL(t *testing.T) {
	mockLoader := new(loaderMock)
	mockLoader.On("GetSourceURL").Return(nil)
	mockLoader.On("GetReader", mock.Anything).Return(io.NopCloser(bytes.NewReader(wasmdata.TestModule)), nil)

	eval, err := FromExtismLoader(t.Context(), mockLoader, WithEntryPoint(wasmdata.EntrypointGreet))
	require.NoError(t, err)
	require.NotNil(t, eval)
	mockLoader.AssertExpectations(t)
}

func TestFromExtismLoader_DiskLoader(t *testing.T) {
	tmpDir := t.TempDir()
	tempFilePath := fmt.Sprintf("%s/test.wasm", tmpDir)
	require.NoError(t, os.WriteFile(tempFilePath, wasmdata.TestModule, 0o600))

	diskLoader, err := loader.NewFromDisk(tempFilePath)
	require.NoError(t, err)
	require.NotNil(t, diskLoader)

	eval, err := FromExtismLoader(t.Context(), diskLoader, WithEntryPoint(wasmdata.EntrypointGreet))
	require.NoError(t, err)
	require.NotNil(t, eval)
	assert.Equal(t, "extism.Evaluator", eval.String())
}

func TestFromExtismLoader_RunsEndToEnd(t *testing.T) {
	scriptLoader, err := loader.NewFromBytes(wasmdata.TestModule)
	require.NoError(t, err)

	eval, err := FromExtismLoader(
		t.Context(),
		scriptLoader,
		WithEntryPoint(wasmdata.EntrypointGreet),
		WithStaticData(map[string]any{"input": "World"}),
	)
	require.NoError(t, err)
	require.NotNil(t, eval)

	res, err := eval.Eval(t.Context())
	require.NoError(t, err)
	got, err := res.AsMap()
	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", got["greeting"])
}

// Mutates slog.Default — must not call t.Parallel(). Cross-package safe
// because `go test ./...` runs each package in its own process.
func TestFromExtismLoader_DefaultsToSlogDefault(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	scriptLoader, err := loader.NewFromBytes(wasmdata.TestModule)
	require.NoError(t, err)

	eval, err := FromExtismLoader(t.Context(), scriptLoader, WithEntryPoint(wasmdata.EntrypointGreet))
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestNewCompiler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		comp, err := NewCompiler(compiler.WithLogHandler(slog.NewTextHandler(os.Stdout, nil)))
		require.NoError(t, err)
		require.NotNil(t, comp)
	})

	t.Run("with multiple options", func(t *testing.T) {
		comp, err := NewCompiler(
			compiler.WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
			compiler.WithEntryPoint("process"),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)
	})
}

// TestFromExtismLoader_LoaderCancelled verifies that ctx cancellation
// during the loader IO phase aborts the constructor. This is the
// headline behavior of #145 for engines whose WASM compile is too fast
// to reliably observe a pre-cancelled ctx (the wazero compile of the
// small embedded test module completes in microseconds).
func TestFromExtismLoader_LoaderCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // pre-cancel; the loader's GetReader call observes it.

	mockLoader := new(loaderMock)
	mockLoader.On("GetSourceURL").Return((*url.URL)(nil))
	mockLoader.On("GetReader", mock.Anything).Return(nil, ctx.Err())

	eval, err := FromExtismLoader(ctx, mockLoader, WithEntryPoint(wasmdata.EntrypointGreet))
	require.Error(t, err)
	require.Nil(t, eval)
	require.ErrorIs(t, err, context.Canceled)
}
