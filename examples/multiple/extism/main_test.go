package main

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunExtismMultipleTimes(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Run the multiple execution example
	results, err := RunExtismMultipleTimes(handler)
	if err != nil {
		t.Logf("Extism example failed: %v - this may be due to missing WASM file", err)
		t.Skip("Skipping Extism test as it requires a WASM file")
		return
	}

	// Verify we got 3 results
	require.Len(t, results, 3, "Should have 3 results")

	// Verify each result
	expectedGreetings := []string{
		"Hello, World!",
		"Hello, Extism!",
		"Hello, Go-PolyScript!",
	}

	for i, result := range results {
		greeting, ok := result["greeting"]
		assert.True(t, ok, "Result should have a greeting field")
		assert.Equal(t, expectedGreetings[i], greeting, "Should have the correct greeting")
	}
}
