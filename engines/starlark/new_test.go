package starlark

import (
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"testing"

	"github.com/robbyt/go-polyscript/engines/starlark/compiler"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testStarlarkScript = `
# Simple Starlark script that prints a message
print("Hello from Starlark")

# Define and call a simple function
def greet(name):
    return "Hello, " + name

result = greet("World")
`

// Helper function to create a string loader with test script
func createTestLoader(t *testing.T) *loader.FromString {
	t.Helper()
	stringLoader, err := loader.NewFromString(testStarlarkScript)
	require.NoError(t, err)
	require.NotNil(t, stringLoader)
	return stringLoader
}

func TestFromStarlarkLoader(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		stringLoader := createTestLoader(t)

		// Execute
		evalInstance, err := FromStarlarkLoader(handler, stringLoader)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evalInstance)
		assert.Equal(t, "starlark.Evaluator", evalInstance.String())
	})

	t.Run("error from loader", func(t *testing.T) {
		// Setup - create a mock loader that will return an error
		handler := slog.NewTextHandler(os.Stdout, nil)
		mockLoader := new(loader.MockLoader)
		mockURL, err := url.Parse("file:///test-starlark-file.star")
		require.NoError(t, err, "Failed to parse URL")
		mockLoader.On("GetSourceURL").Return(mockURL)
		mockLoader.On("GetReader").Return(nil, fmt.Errorf("failed to load script"))

		// Execute
		evalInstance, err := FromStarlarkLoader(handler, mockLoader)

		// Verify
		require.Error(t, err)
		require.Nil(t, evalInstance)
		assert.Contains(t, err.Error(), "failed to load script")
		mockLoader.AssertExpectations(t)
	})
}

func TestFromStarlarkLoaderWithData(t *testing.T) {
	t.Parallel()

	t.Run("success with static data", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		stringLoader := createTestLoader(t)

		staticData := map[string]any{
			"version": "1.0.0",
			"config": map[string]any{
				"timeout": 30,
				"retry":   true,
			},
		}

		// Execute
		evalInstance, err := FromStarlarkLoaderWithData(handler, stringLoader, staticData)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evalInstance)
		assert.Equal(t, "starlark.Evaluator", evalInstance.String())
	})

	t.Run("empty static data", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		stringLoader := createTestLoader(t)

		// Execute
		evalInstance, err := FromStarlarkLoaderWithData(handler, stringLoader, map[string]any{})

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evalInstance)
		assert.Equal(t, "starlark.Evaluator", evalInstance.String())
	})

	t.Run("error from loader", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		mockLoader := new(loader.MockLoader)
		mockURL, err := url.Parse("file:///test-starlark-file.star")
		require.NoError(t, err, "Failed to parse URL")
		mockLoader.On("GetSourceURL").Return(mockURL)
		mockLoader.On("GetReader").Return(nil, fmt.Errorf("failed to load script"))
		staticData := map[string]any{"version": "1.0.0"}

		// Execute
		evalInstance, err := FromStarlarkLoaderWithData(handler, mockLoader, staticData)

		// Verify
		require.Error(t, err)
		require.Nil(t, evalInstance)
		assert.Contains(t, err.Error(), "failed to load script")
		mockLoader.AssertExpectations(t)
	})
}

func TestNewCompiler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Execute
		comp, err := NewCompiler(compiler.WithLogHandler(slog.NewTextHandler(os.Stdout, nil)))

		// Verify
		require.NoError(t, err)
		require.NotNil(t, comp)
	})

	t.Run("with multiple options", func(t *testing.T) {
		// Execute
		comp, err := NewCompiler(
			compiler.WithLogHandler(slog.NewTextHandler(os.Stdout, nil)),
			compiler.WithGlobals([]string{"data", "context"}),
		)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, comp)
	})
}

func TestNewEvaluator(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		stringLoader := createTestLoader(t)
		provider := data.NewContextProvider(constants.EvalData)

		// Execute
		evalInstance, err := NewEvaluator(
			handler,
			stringLoader,
			provider,
		)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evalInstance)
		assert.Equal(t, "starlark.Evaluator", evalInstance.String())
	})

	t.Run("with nil handler", func(t *testing.T) {
		// Setup
		stringLoader := createTestLoader(t)
		provider := data.NewContextProvider(constants.EvalData)

		// Execute
		evalInstance, err := NewEvaluator(
			nil,
			stringLoader,
			provider,
		)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evalInstance)
		assert.Equal(t, "starlark.Evaluator", evalInstance.String())
	})

	t.Run("loader error", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		mockLoader := new(loader.MockLoader)
		mockURL, err := url.Parse("file:///test-starlark-file.star")
		require.NoError(t, err, "Failed to parse URL")
		mockLoader.On("GetSourceURL").Return(mockURL)
		mockLoader.On("GetReader").Return(nil, fmt.Errorf("failed to load content"))
		provider := data.NewContextProvider(constants.EvalData)

		// Execute
		evalInstance, err := NewEvaluator(
			handler,
			mockLoader,
			provider,
		)

		// Verify
		require.Error(t, err)
		require.Nil(t, evalInstance)
		assert.Contains(t, err.Error(), "failed to load content")
		mockLoader.AssertExpectations(t)
	})

	t.Run("nil provider", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		stringLoader := createTestLoader(t)

		// Execute
		evalInstance, err := NewEvaluator(
			handler,
			stringLoader,
			nil,
		)

		// Verify
		require.Error(t, err)
		require.Nil(t, evalInstance)
		require.Contains(t, err.Error(), "provider is nil")
	})

	t.Run("invalid script syntax", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		invalidScript := `this is { not valid } Starlark syntax`
		invalidLoader, err := loader.NewFromString(invalidScript)
		require.NoError(t, err)
		provider := data.NewContextProvider(constants.EvalData)

		// Execute
		evalInstance, err := NewEvaluator(
			handler,
			invalidLoader,
			provider,
		)

		// Verify
		require.Error(t, err)
		require.Nil(t, evalInstance)
		// Update error message check to match actual Starlark error message
		assert.ErrorIs(t, err, compiler.ErrValidationFailed)
	})
}

func TestDiskLoaderIntegration(t *testing.T) {
	t.Run("create from disk loader", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)

		// write test script to tmp file, load it
		tmpDir := t.TempDir()
		tempFilePath := fmt.Sprintf("%s/test.star", tmpDir)
		err := os.WriteFile(tempFilePath, []byte(testStarlarkScript), 0o644)
		require.NoError(t, err)

		// Create a disk loader for the temporary file
		diskLoader, err := loader.NewFromDisk(tempFilePath)
		require.NoError(t, err)
		require.NotNil(t, diskLoader)

		provider := data.NewContextProvider(constants.EvalData)

		// Execute
		evalInstance, err := NewEvaluator(
			handler,
			diskLoader,
			provider,
		)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evalInstance)
		assert.Equal(t, "starlark.Evaluator", evalInstance.String())

		// Verify the disk loader has correct path
		fileURL := diskLoader.GetSourceURL()
		require.NotNil(t, fileURL)
		assert.Contains(t, fileURL.String(), "test.star")

		// Verify content was loaded correctly
		reader, err := diskLoader.GetReader()
		require.NoError(t, err)
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Equal(t, testStarlarkScript, string(content))

		// Properly close the reader when done
		err = reader.Close()
		require.NoError(t, err, "Failed to close reader")
	})
}
