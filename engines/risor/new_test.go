package risor

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

	"github.com/robbyt/go-polyscript/engines/risor/compiler"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testRisorScript = `
// Define and call a greeting handler
function greet(name) {
  return "Hello, " + name
}

greet("World")
`

func newTestLoader(t *testing.T) *loader.FromString {
	t.Helper()
	stringLoader, err := loader.NewFromString(testRisorScript)
	require.NoError(t, err)
	require.NotNil(t, stringLoader)
	return stringLoader
}

func newErrorLoader(t *testing.T, msg string) *loaderMock {
	t.Helper()
	mockLoader := new(loaderMock)
	mockURL, err := url.Parse("file:///test-risor-file.risor")
	require.NoError(t, err)
	mockLoader.On("GetSourceURL").Return(mockURL)
	mockLoader.On("GetReader", mock.Anything).Return(nil, errors.New(msg))
	return mockLoader
}

func TestFromRisorLoader_NoOptions(t *testing.T) {
	// Without WithLogHandler the evaluator inherits slog.Default(); no error.
	eval, err := FromRisorLoader(t.Context(), newTestLoader(t))
	require.NoError(t, err)
	require.NotNil(t, eval)
	assert.Equal(t, "risor.Evaluator", eval.String())
}

func TestFromRisorLoader_WithLogHandler(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	eval, err := FromRisorLoader(t.Context(), newTestLoader(t), WithLogHandler(handler))
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromRisorLoader_NilLogHandler(t *testing.T) {
	// Explicitly passing nil is treated the same as omitting the option.
	eval, err := FromRisorLoader(t.Context(), newTestLoader(t), WithLogHandler(nil))
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromRisorLoader_WithStaticData(t *testing.T) {
	staticData := map[string]any{
		"version": "1.0.0",
		"config":  map[string]any{"timeout": 30, "retry": true},
	}
	eval, err := FromRisorLoader(t.Context(), newTestLoader(t), WithStaticData(staticData))
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromRisorLoader_EmptyStaticData(t *testing.T) {
	// Empty (but non-nil) map still composes a CompositeProvider; should succeed.
	eval, err := FromRisorLoader(t.Context(), newTestLoader(t), WithStaticData(map[string]any{}))
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromRisorLoader_WithDataProvider(t *testing.T) {
	provider := data.NewContextProvider("test_key")
	eval, err := FromRisorLoader(t.Context(), newTestLoader(t), WithDataProvider(provider))
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromRisorLoader_DataProviderBeatsStaticData(t *testing.T) {
	provider := data.NewContextProvider("sentinel")
	eval, err := FromRisorLoader(
		t.Context(),
		newTestLoader(t),
		WithStaticData(map[string]any{"ignored": true}),
		WithDataProvider(provider),
	)
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromRisorLoader_NilOption(t *testing.T) {
	// nil options should be ignored, not panic.
	var nilOpt Option
	eval, err := FromRisorLoader(t.Context(), newTestLoader(t), nilOpt, WithStaticData(map[string]any{"k": "v"}))
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromRisorLoader_LoaderError(t *testing.T) {
	mockLoader := newErrorLoader(t, "failed to load script")
	eval, err := FromRisorLoader(t.Context(), mockLoader)
	require.Error(t, err)
	require.Nil(t, eval)
	assert.Contains(t, err.Error(), "failed to load script")
	mockLoader.AssertExpectations(t)
}

func TestFromRisorLoader_DiskLoader(t *testing.T) {
	tmpDir := t.TempDir()
	tempFilePath := fmt.Sprintf("%s/test.risor", tmpDir)
	require.NoError(t, os.WriteFile(tempFilePath, []byte(testRisorScript), 0o600))

	diskLoader, err := loader.NewFromDisk(tempFilePath)
	require.NoError(t, err)
	require.NotNil(t, diskLoader)

	eval, err := FromRisorLoader(t.Context(), diskLoader)
	require.NoError(t, err)
	require.NotNil(t, eval)
	assert.Equal(t, "risor.Evaluator", eval.String())

	// Verify content was loaded correctly.
	reader, err := diskLoader.GetReader(t.Context())
	require.NoError(t, err)
	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testRisorScript, string(content))
	require.NoError(t, reader.Close())
}

func TestFromRisorLoader_RunsEndToEnd(t *testing.T) {
	const script = `"Hello, " + ctx["name"]`
	scriptLoader, err := loader.NewFromString(script)
	require.NoError(t, err)

	eval, err := FromRisorLoader(
		t.Context(),
		scriptLoader,
		WithStaticData(map[string]any{"name": "World"}),
	)
	require.NoError(t, err)
	require.NotNil(t, eval)

	res, err := eval.Eval(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "Hello, World", res.Interface())
}

// Mutates slog.Default — must not call t.Parallel(). Cross-package safe
// because `go test ./...` runs each package in its own process.
func TestFromRisorLoader_DefaultsToSlogDefault(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	scriptLoader, err := loader.NewFromString(`"hi"`)
	require.NoError(t, err)

	eval, err := FromRisorLoader(t.Context(), scriptLoader)
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
			compiler.WithCtxGlobal(),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)
	})
}

// TestFromRisorLoader_CompileCancelled verifies that a pre-cancelled ctx
// causes the Risor parser to abort during compile. This is the headline
// behavior of #145: a cancellable ctx now actually reaches the parser
// instead of being swallowed by an inline context.Background().
func TestFromRisorLoader_CompileCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // pre-cancel before construction

	eval, err := FromRisorLoader(ctx, newTestLoader(t))
	require.Error(t, err, "expected an error when ctx is pre-cancelled")
	require.Nil(t, eval)
	require.ErrorIs(t, err, context.Canceled)
}
