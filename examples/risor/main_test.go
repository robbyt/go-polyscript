package risor

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRisorExample(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Run the example
	result, err := RunRisorExample(handler)
	require.NoError(t, err, "Risor example should run without error")

	// Verify the result
	assert.Equal(t, "Hello, World!", result["greeting"], "Should have the correct greeting")

	// Check length based on its type (could be float64 or int64 depending on implementation)
	length := result["length"]
	assert.NotNil(t, length, "Should have a length field")
	lengthValue, ok := length.(int64)
	if !ok {
		lengthValueFloat, ok := length.(float64)
		if ok {
			assert.Equal(t, float64(13), lengthValueFloat, "Should have the correct length")
		} else {
			assert.Fail(t, "Length is neither int64 nor float64")
		}
	} else {
		assert.Equal(t, int64(13), lengthValue, "Should have the correct length")
	}
}
