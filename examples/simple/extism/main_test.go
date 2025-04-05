package main

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunExtismExample(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Run the example
	result, err := runExtismExample(handler)
	require.NoError(t, err, "Extism example should run without error")

	// Verify the result
	assert.NotEmpty(t, result, "Result should not be empty")
	assert.Contains(t, result, "greeting", "Result should contain a greeting field")
}

func TestRun(t *testing.T) {
	assert.NoError(t, run(), "run() should execute without errors")
}
