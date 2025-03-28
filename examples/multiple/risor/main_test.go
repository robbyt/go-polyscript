package main

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRisorExampleMultipleTimes(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Run the example
	results, err := RunRisorExampleMultipleTimes(handler)
	require.NoError(t, err, "Risor example should run without error")

	// Verify we got the expected number of results
	require.Len(t, results, 3, "Should have 3 results")

	// Expected greetings based on the inputs
	expectedGreetings := []string{
		"Hello, World!",
		"Hello, Risor!",
		"Hello, Go!",
	}

	// Expected lengths based on the greetings
	expectedLengths := []int64{
		13, // length of "Hello, World!"
		13, // length of "Hello, Risor!"
		10, // length of "Hello, Go!"
	}

	// Verify each result
	for i, result := range results {
		// Verify greeting
		greeting, ok := result["greeting"].(string)
		assert.True(t, ok, "Greeting should be a string")
		assert.Equal(t, expectedGreetings[i], greeting, "Should have the correct greeting")

		// Verify length
		length := result["length"]
		assert.NotNil(t, length, "Should have a length field")

		// Check length based on its type (could be float64 or int64 depending on implementation)
		lengthValue, ok := length.(int64)
		if !ok {
			lengthValueFloat, ok := length.(float64)
			if ok {
				assert.Equal(
					t,
					float64(expectedLengths[i]),
					lengthValueFloat,
					"Should have the correct length",
				)
			} else {
				assert.Fail(t, "Length is neither int64 nor float64")
			}
		} else {
			assert.Equal(t, expectedLengths[i], lengthValue, "Should have the correct length")
		}
	}
}
