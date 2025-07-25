package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunExtismExample(t *testing.T) {
	result, err := runExtismExample(nil)
	require.NoError(t, err, "runExtismExample should not return an error")
	require.NotNil(t, result, "Result should not be nil")

	greeting := result["greeting"]
	require.IsType(t, "", greeting, "Greeting should be a string")
	assert.Equal(t, "Hello, World!", greeting, "Should have the correct greeting")
}

func TestRun(t *testing.T) {
	err := run()
	require.NoError(t, err, "run() should execute without error")
}
