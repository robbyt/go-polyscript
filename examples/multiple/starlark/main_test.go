package main

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunStarlarkExampleMultipleTimes(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Run the "compile once, run many times" example
	results, err := RunStarlarkExampleMultipleTimes(handler)
	require.NoError(t, err, "Multiple executions should run without error")

	// Expect 4 results (World, Alice, Bob, Charlie)
	require.Len(t, results, 4, "Should have 4 results from multiple executions")

	// Check each result
	expectedNames := []string{"World", "Alice", "Bob", "Charlie"}
	for i, name := range expectedNames {
		result := results[i]
		expectedGreeting := "Hello, " + name + "!"

		// Verify greeting
		assert.Equal(t, expectedGreeting, result["greeting"], "Should have the correct greeting for %s", name)

		// Verify length
		length := result["length"]
		assert.NotNil(t, length, "Should have a length field")

		// Length should match the greeting length
		expectedLength := int64(len(expectedGreeting))

		// Handle different numeric types
		lengthValue, ok := length.(int64)
		if !ok {
			lengthValueFloat, ok := length.(float64)
			if ok {
				assert.Equal(t, float64(expectedLength), lengthValueFloat, "Should have the correct length for %s", name)
			} else {
				assert.Fail(t, "Length is neither int64 nor float64")
			}
		} else {
			assert.Equal(t, expectedLength, lengthValue, "Should have the correct length for %s", name)
		}
	}
}
