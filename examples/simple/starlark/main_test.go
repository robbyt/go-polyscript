package main

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunStarlarkExample(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Run the example
	result, err := runStarlarkExample(handler)
	require.NoError(t, err, "Starlark example should run without error")

	// Verify the result
	assert.Equal(t, "Hello, World!", result["greeting"], "Should have the correct greeting")
	assert.Equal(t, int64(13), result["length"], "Should have the correct length")
}

func TestRun(t *testing.T) {
	assert.NoError(t, run(), "run() should execute without errors")
}
