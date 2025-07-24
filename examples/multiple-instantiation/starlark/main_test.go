package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunMultipleTimes(t *testing.T) {
	results, err := runMultipleTimes(nil)
	require.NoError(t, err, "runMultipleTimes should not return an error")
	require.NotNil(t, results, "Results should not be nil")

	expectedResults := []struct {
		name     string
		greeting string
		length   int64
	}{
		{
			name:     "World",
			greeting: "Hello, World!",
			length:   13,
		},
		{
			name:     "Alice",
			greeting: "Hello, Alice!",
			length:   13,
		},
		{
			name:     "Bob",
			greeting: "Hello, Bob!",
			length:   11,
		},
		{
			name:     "Charlie",
			greeting: "Hello, Charlie!",
			length:   15,
		},
	}

	require.Len(t, results, len(expectedResults), "Should have %d results", len(expectedResults))

	for i, expected := range expectedResults {
		t.Run(expected.name, func(t *testing.T) {
			result := results[i]
			require.NotNil(t, result, "Result at index %d should not be nil", i)

			greeting, exists := result["greeting"]
			require.True(t, exists, "Result should have a greeting field")
			require.IsType(t, "", greeting, "Greeting should be a string")
			assert.Equal(t, expected.greeting, greeting, "Should have the correct greeting")

			length, exists := result["length"]
			require.True(t, exists, "Result should have a length field")
			require.IsType(t, int64(0), length, "Length should be int64")
			assert.Equal(t, expected.length, length, "Should have the correct length")
		})
	}
}

func TestRun(t *testing.T) {
	err := run()
	require.NoError(t, err, "run() should execute without error")
}
