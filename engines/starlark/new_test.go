package starlark

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"testing"

	"github.com/robbyt/go-polyscript/engines/starlark/compiler"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testStarlarkScript = `
# Define and call a simple function
def greet(name):
    return "Hello, " + name

result = greet("World")
_ = result
`

func newTestLoader(t *testing.T) *loader.FromString {
	t.Helper()
	stringLoader, err := loader.NewFromString(testStarlarkScript)
	require.NoError(t, err)
	require.NotNil(t, stringLoader)
	return stringLoader
}

func newErrorLoader(t *testing.T, msg string) *loaderMock {
	t.Helper()
	mockLoader := new(loaderMock)
	mockURL, err := url.Parse("file:///test-starlark-file.star")
	require.NoError(t, err)
	mockLoader.On("GetSourceURL").Return(mockURL)
	mockLoader.On("GetReader", mock.Anything).Return(nil, errors.New(msg))
	return mockLoader
}

func TestFromStarlarkLoader_NoOptions(t *testing.T) {
	eval, err := FromStarlarkLoader(newTestLoader(t))
	require.NoError(t, err)
	require.NotNil(t, eval)
	assert.Equal(t, "starlark.Evaluator", eval.String())
}

func TestFromStarlarkLoader_WithLogHandler(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	eval, err := FromStarlarkLoader(newTestLoader(t), WithLogHandler(handler))
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromStarlarkLoader_NilLogHandler(t *testing.T) {
	eval, err := FromStarlarkLoader(newTestLoader(t), WithLogHandler(nil))
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromStarlarkLoader_WithStaticData(t *testing.T) {
	staticData := map[string]any{
		"version": "1.0.0",
		"config":  map[string]any{"timeout": 30, "retry": true},
	}
	eval, err := FromStarlarkLoader(newTestLoader(t), WithStaticData(staticData))
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromStarlarkLoader_EmptyStaticData(t *testing.T) {
	eval, err := FromStarlarkLoader(newTestLoader(t), WithStaticData(map[string]any{}))
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromStarlarkLoader_WithDataProvider(t *testing.T) {
	provider := data.NewContextProvider("test_key")
	eval, err := FromStarlarkLoader(newTestLoader(t), WithDataProvider(provider))
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromStarlarkLoader_DataProviderBeatsStaticData(t *testing.T) {
	provider := data.NewContextProvider("sentinel")
	eval, err := FromStarlarkLoader(
		newTestLoader(t),
		WithStaticData(map[string]any{"ignored": true}),
		WithDataProvider(provider),
	)
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromStarlarkLoader_NilOption(t *testing.T) {
	var nilOpt Option
	eval, err := FromStarlarkLoader(newTestLoader(t), nilOpt, WithStaticData(map[string]any{"k": "v"}))
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestFromStarlarkLoader_LoaderError(t *testing.T) {
	mockLoader := newErrorLoader(t, "failed to load script")
	eval, err := FromStarlarkLoader(mockLoader)
	require.Error(t, err)
	require.Nil(t, eval)
	assert.Contains(t, err.Error(), "failed to load script")
	mockLoader.AssertExpectations(t)
}

func TestFromStarlarkLoader_InvalidScript(t *testing.T) {
	invalidLoader, err := loader.NewFromString(`this is { not valid } Starlark syntax`)
	require.NoError(t, err)
	eval, err := FromStarlarkLoader(invalidLoader)
	require.Error(t, err)
	require.Nil(t, eval)
	assert.ErrorIs(t, err, compiler.ErrValidationFailed)
}

func TestFromStarlarkLoader_DiskLoader(t *testing.T) {
	tmpDir := t.TempDir()
	tempFilePath := fmt.Sprintf("%s/test.star", tmpDir)
	require.NoError(t, os.WriteFile(tempFilePath, []byte(testStarlarkScript), 0o600))

	diskLoader, err := loader.NewFromDisk(tempFilePath)
	require.NoError(t, err)
	require.NotNil(t, diskLoader)

	eval, err := FromStarlarkLoader(diskLoader)
	require.NoError(t, err)
	require.NotNil(t, eval)
	assert.Equal(t, "starlark.Evaluator", eval.String())

	reader, err := diskLoader.GetReader(t.Context())
	require.NoError(t, err)
	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testStarlarkScript, string(content))
	require.NoError(t, reader.Close())
}

func TestFromStarlarkLoader_RunsEndToEnd(t *testing.T) {
	const script = `
result = "Hello, " + ctx["name"]
_ = result
`
	scriptLoader, err := loader.NewFromString(script)
	require.NoError(t, err)

	eval, err := FromStarlarkLoader(
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
func TestFromStarlarkLoader_DefaultsToSlogDefault(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	scriptLoader, err := loader.NewFromString(`_ = "ok"`)
	require.NoError(t, err)

	eval, err := FromStarlarkLoader(scriptLoader)
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
			compiler.WithGlobals([]string{"data", "context"}),
		)
		require.NoError(t, err)
		require.NotNil(t, comp)
	})
}
