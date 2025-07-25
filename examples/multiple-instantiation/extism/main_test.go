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
	require.Len(t, results, 3, "Should have 3 results")

	expectedResults := []string{
		"Hello, World!",
		"Hello, Extism!",
		"Hello, Go-PolyScript!",
	}

	for i, result := range results {
		assert.NotNil(t, result, "Result at index %d should not be nil", i)

		greeting := result["greeting"]
		require.IsType(t, "", greeting, "Greeting at index %d should be a string", i)
		assert.Equal(t, expectedResults[i], greeting,
			"Result at index %d should have the correct greeting", i,
		)
	}
}

func TestRun(t *testing.T) {
	err := run()
	require.NoError(t, err, "run() should execute without error")
}
