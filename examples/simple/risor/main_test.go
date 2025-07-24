package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRisorExample(t *testing.T) {
	result, err := runRisorExample(nil)
	require.NoError(t, err, "runRisorExample should not return an error")
	require.NotNil(t, result, "Result should not be nil")

	greeting, exists := result["greeting"]
	require.True(t, exists, "Result should have a greeting field")
	require.IsType(t, "", greeting, "Greeting should be a string")
	assert.Equal(t, "Hello, World!", greeting, "Should have the correct greeting")

	length, exists := result["length"]
	require.True(t, exists, "Result should have a length field")
	require.IsType(t, int64(0), length, "Length should be int64")
	assert.Equal(t, int64(13), length, "Should have the correct length")
}

func TestRun(t *testing.T) {
	err := run()
	require.NoError(t, err, "run() should execute without error")
}
