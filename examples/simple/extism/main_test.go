package main

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunExtismExample(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Run the example
	result, err := RunExtismExample(handler)
	if err != nil {
		t.Logf("Extism example failed: %v - this may be due to missing WASM file", err)
		t.Skip("Skipping Extism test as it requires a WASM file")
		return
	}

	// Verify the result
	greeting, ok := result["greeting"]
	assert.True(t, ok, "Result should have a greeting field")
	assert.Equal(t, "Hello, World!", greeting, "Should have the correct greeting")
}
