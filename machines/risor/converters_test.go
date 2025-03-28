package risor

import (
	"testing"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/stretchr/testify/require"
)

// TestConvertToRisorOptions tests the convertToRisorOptions method
func TestConvertToRisorOptions(t *testing.T) {
	t.Parallel()

	// Test with empty data
	options := convertToRisorOptions(constants.Ctx, map[string]any{})
	require.Len(t, options, 1)

	// Test with actual data
	testData := map[string]any{"foo": "bar"}
	options = convertToRisorOptions(constants.Ctx, testData)
	require.Len(t, options, 1)
}
