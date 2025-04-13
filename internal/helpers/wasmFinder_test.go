package helpers

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindWasmFile(t *testing.T) {
	t.Parallel()

	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	t.Run("nonexistent file returns error", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := os.Chdir(tmpDir)
		require.NoError(t, err)

		// Try to find a nonexistent WASM file
		foundPath, err := FindWasmFile(logger)
		assert.Empty(t, foundPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "WASM file not found")
		assert.Contains(t, err.Error(), tmpDir) // Should mention the searched paths
	})

	t.Run("found file returns path", func(t *testing.T) {
		// Create a test file
		tmpDir := t.TempDir()
		wasmPath := filepath.Join(tmpDir, "main.wasm")
		err := os.WriteFile(wasmPath, []byte("dummy wasm content"), 0o644)
		require.NoError(t, err)

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Try to find the WASM file
		foundPath, err := FindWasmFile(logger)
		assert.NoError(t, err)
		assert.True(t, strings.HasSuffix(foundPath, "main.wasm"))
	})
}
