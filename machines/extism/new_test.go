package extism

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/machines/extism/compiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestWasmBytes returns the test WASM bytes from the examples directory
func getTestWasmBytes(t *testing.T) []byte {
	t.Helper()
	// Find the main.wasm file in the examples directory
	wasmPath := filepath.Join("..", "..", "examples", "testdata", "main.wasm")
	bytes, err := os.ReadFile(wasmPath)
	require.NoError(t, err, "Failed to read test WASM file")
	require.NotEmpty(t, bytes, "Test WASM file is empty")
	return bytes
}

func setupMockLoader(t *testing.T) *loader.MockLoader {
	t.Helper()
	mockLoader := new(loader.MockLoader)
	mockURL, err := url.Parse("file:///test-wasm-file.wasm")
	require.NoError(t, err, "Failed to parse URL")
	mockLoader.On("GetSourceURL").Return(mockURL)

	// Create a reader that will call Close on the mock loader when it's closed
	wasmBytes := getTestWasmBytes(t)
	reader := io.NopCloser(bytes.NewReader(wasmBytes))
	mockLoader.On("GetReader").Return(reader, nil)

	// We don't expect Close to be called directly on the loader,
	// it seems the code doesn't call it directly
	return mockLoader
}

func setupErrorMockLoader(t *testing.T) *loader.MockLoader {
	t.Helper()
	mockLoader := new(loader.MockLoader)
	mockURL, err := url.Parse("file:///test-wasm-file.wasm")
	require.NoError(t, err, "Failed to parse URL")
	mockLoader.On("GetSourceURL").Return(mockURL)
	mockLoader.On("GetReader").Return(nil, fmt.Errorf("failed to load WASM"))
	// Don't expect Close for error case
	return mockLoader
}

func TestFromExtismLoader(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		mockLoader := setupMockLoader(t)

		// Execute
		evaluator, err := FromExtismLoader(handler, mockLoader, "greet")

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evaluator)
		assert.Equal(t, "extism.Evaluator", evaluator.String())
		mockLoader.AssertExpectations(t)
	})

	t.Run("error from loader", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		mockLoader := setupErrorMockLoader(t)

		// Execute
		evaluator, err := FromExtismLoader(handler, mockLoader, "greet")

		// Verify
		require.Error(t, err)
		require.Nil(t, evaluator)
		mockLoader.AssertExpectations(t)
	})

	t.Run("nil URL in loader", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		mockLoader := new(loader.MockLoader)
		mockLoader.On("GetSourceURL").Return(nil)
		mockLoader.On("GetReader").Return(io.NopCloser(bytes.NewReader(getTestWasmBytes(t))), nil)
		// Don't expect Close - loader.Close() is not called by the code

		// Execute
		evaluator, err := FromExtismLoader(handler, mockLoader, "greet")

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evaluator)
		mockLoader.AssertExpectations(t)
	})
}

func TestFromExtismLoaderWithData(t *testing.T) {
	t.Parallel()

	t.Run("success with static data", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		mockLoader := setupMockLoader(t)

		staticData := map[string]any{
			"version": "1.0.0",
			"config": map[string]any{
				"timeout": 30,
				"retry":   true,
			},
		}

		// Execute
		evaluator, err := FromExtismLoaderWithData(handler, mockLoader, staticData, "greet")

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evaluator)
		mockLoader.AssertExpectations(t)
	})

	t.Run("empty static data", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		mockLoader := setupMockLoader(t)

		// Execute
		evaluator, err := FromExtismLoaderWithData(handler, mockLoader, map[string]any{}, "greet")

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evaluator)
		mockLoader.AssertExpectations(t)
	})

	t.Run("error from loader", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		mockLoader := setupErrorMockLoader(t)

		staticData := map[string]any{"version": "1.0.0"}

		// Execute
		evaluator, err := FromExtismLoaderWithData(handler, mockLoader, staticData, "greet")

		// Verify
		require.Error(t, err)
		require.Nil(t, evaluator)
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
			compiler.WithEntryPoint("process"),
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
		mockLoader := setupMockLoader(t)
		provider := data.NewContextProvider("test_key")

		// Execute
		evalInstance, err := NewEvaluator(
			handler,
			mockLoader,
			provider,
			"greet",
		)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evalInstance)
		assert.Equal(t, "extism.Evaluator", evalInstance.String())
		mockLoader.AssertExpectations(t)
	})

	t.Run("with nil URL", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		mockLoader := new(loader.MockLoader)
		mockLoader.On("GetSourceURL").Return(nil)
		mockLoader.On("GetReader").Return(io.NopCloser(bytes.NewReader(getTestWasmBytes(t))), nil)
		// Don't expect Close - loader.Close() is not called by the code

		provider := data.NewContextProvider("test_key")

		// Execute
		evaluator, err := NewEvaluator(
			handler,
			mockLoader,
			provider,
			"greet",
		)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evaluator)
		mockLoader.AssertExpectations(t)
	})

	t.Run("with nil handler", func(t *testing.T) {
		// Setup
		mockLoader := setupMockLoader(t)
		provider := data.NewContextProvider("test_key")

		// Execute
		evaluator, err := NewEvaluator(
			nil,
			mockLoader,
			provider,
			"greet",
		)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evaluator)
		mockLoader.AssertExpectations(t)
	})

	t.Run("loader error", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		mockLoader := setupErrorMockLoader(t)
		provider := data.NewContextProvider("test_key")

		// Execute
		evaluator, err := NewEvaluator(
			handler,
			mockLoader,
			provider,
			"greet",
		)

		// Verify
		require.Error(t, err)
		require.Nil(t, evaluator)
		mockLoader.AssertExpectations(t)
	})

	t.Run("nil provider", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		// Don't use setupMockLoader for this test since it won't be used
		mockLoader := new(loader.MockLoader)

		// Execute
		evaluator, err := NewEvaluator(
			handler,
			mockLoader,
			nil,
			"greet",
		)

		// Verify
		require.Error(t, err)
		require.Nil(t, evaluator)
		require.Contains(t, err.Error(), "provider is nil")
	})
}

func TestDiskLoaderIntegration(t *testing.T) {
	t.Run("create from disk loader", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)

		// Create a temporary directory
		tmpDir := t.TempDir()

		// Get WASM bytes for test
		wasmBytes := getTestWasmBytes(t)

		// Create a temporary file in the temporary directory
		tempFilePath := fmt.Sprintf("%s/test.wasm", tmpDir)
		err := os.WriteFile(tempFilePath, wasmBytes, 0o644)
		require.NoError(t, err)

		// Create a disk loader for the temporary file
		diskLoader, err := loader.NewFromDisk(tempFilePath)
		require.NoError(t, err)
		require.NotNil(t, diskLoader)

		provider := data.NewContextProvider("test_key")

		// Execute
		evaluator, err := NewEvaluator(
			handler,
			diskLoader,
			provider,
			"greet",
		)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evaluator)
		assert.Equal(t, "extism.Evaluator", evaluator.String())

		// Verify the disk loader has correct path
		fileURL := diskLoader.GetSourceURL()
		require.NotNil(t, fileURL)
		assert.Contains(t, fileURL.String(), "test.wasm")

		// Verify content was loaded correctly
		reader, err := diskLoader.GetReader()
		require.NoError(t, err)
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Equal(t, wasmBytes, content)
		err = reader.Close() // Close the reader when done
		require.NoError(t, err, "Failed to close reader")
	})
}
