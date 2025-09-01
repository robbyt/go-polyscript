package data

import (
	"log/slog"
	"testing"

	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddDataToContextHelper tests the AddDataToContextHelper utility function
func TestAddDataToContextHelper(t *testing.T) {
	t.Parallel()

	// Create a test logger that discards output
	logger := slog.Default()

	t.Run("nil provider returns error", func(t *testing.T) {
		baseCtx := t.Context()
		enrichedCtx, err := AddDataToContextHelper(
			baseCtx,
			logger,
			nil,
			map[string]any{"key": "value"},
		)

		assert.Error(t, err)
		assert.Equal(t, baseCtx, enrichedCtx, "Context should remain unchanged")
	})

	t.Run("static provider always returns error", func(t *testing.T) {
		provider := NewStaticProvider(simpleData)
		baseCtx := t.Context()

		enrichedCtx, err := AddDataToContextHelper(
			baseCtx,
			logger,
			provider,
			map[string]any{"key": "value"},
		)

		assert.Error(t, err)
		assert.Equal(t, baseCtx, enrichedCtx, "Context should remain unchanged")
		assert.Nil(t, enrichedCtx.Value(constants.EvalData), "Context should not have data added")
	})

	t.Run("context provider with valid data", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		baseCtx := t.Context()

		enrichedCtx, err := AddDataToContextHelper(
			baseCtx,
			logger,
			provider,
			map[string]any{"key": "value"},
		)

		assert.NoError(t, err)
		assert.NotEqual(t, baseCtx, enrichedCtx, "Context should be modified")

		// Verify data was added to context
		data := enrichedCtx.Value(constants.EvalData)
		require.NotNil(t, data)

		contextMap, ok := data.(map[string]any)
		require.True(t, ok, "Context value should be a map")

		// Data should be at root level with no namespace
		assert.Equal(t, "value", contextMap["key"], "Should have key at root level")
	})

	t.Run("context provider with HTTP request", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		baseCtx := t.Context()
		req := createTestRequestHelper()

		// Wrap request in map
		enrichedCtx, err := AddDataToContextHelper(
			baseCtx,
			logger,
			provider,
			map[string]any{"request": req},
		)

		assert.NoError(t, err)
		assert.NotEqual(t, baseCtx, enrichedCtx, "Context should be modified")

		// Verify request data was added to context
		data := enrichedCtx.Value(constants.EvalData)
		require.NotNil(t, data)

		contextMap, ok := data.(map[string]any)
		require.True(t, ok, "Context value should be a map")

		// Data should be under request key
		assert.Contains(t, contextMap, "request")
		requestData, ok := contextMap["request"].(map[string]any)
		require.True(t, ok, "Request data should be a map")
		assert.Equal(t, "GET", requestData["Method"])
		assert.Equal(t, "/test", requestData["URL_Path"])
	})

	t.Run("context provider with mixed data", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		baseCtx := t.Context()
		req := createTestRequestHelper()

		enrichedCtx, err := AddDataToContextHelper(baseCtx, logger, provider,
			map[string]any{"key": "value"}, map[string]any{"request": req})

		assert.NoError(t, err)
		assert.NotEqual(t, baseCtx, enrichedCtx, "Context should be modified")

		// Verify data was added to context
		data := enrichedCtx.Value(constants.EvalData)
		require.NotNil(t, data)

		contextMap, ok := data.(map[string]any)
		require.True(t, ok, "Context value should be a map")

		// Verify map data at root level
		assert.Equal(t, "value", contextMap["key"], "Should have key at root level")

		// Verify request data at root level
		assert.Contains(t, contextMap, "request")
		requestData, ok := contextMap["request"].(map[string]any)
		require.True(t, ok, "Request data should be a map")
		assert.Equal(t, "GET", requestData["Method"])
		assert.Equal(t, "/test", requestData["URL_Path"])
	})

	t.Run("context provider with empty key", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		baseCtx := t.Context()

		enrichedCtx, err := AddDataToContextHelper(
			baseCtx,
			logger,
			provider,
			map[string]any{"": 42}, // Empty key should cause an error
		)

		assert.Error(t, err)
		assert.Equal(t, baseCtx, enrichedCtx, "Context should be unchanged when there's an error")

		// No data should be added to context when there's an error
		data := enrichedCtx.Value(constants.EvalData)
		assert.Nil(t, data, "Context should not have data added when there's an error")
	})

	t.Run("composite provider with mixed success", func(t *testing.T) {
		provider := NewCompositeProvider(
			NewStaticProvider(simpleData),
			NewContextProvider(constants.EvalData),
		)
		baseCtx := t.Context()

		enrichedCtx, err := AddDataToContextHelper(baseCtx, logger, provider,
			map[string]any{"key": "value"})

		assert.NoError(t, err)
		assert.NotEqual(t, baseCtx, enrichedCtx, "Context should be modified")

		// Verify data was added to context
		data := enrichedCtx.Value(constants.EvalData)
		require.NotNil(t, data)

		contextMap, ok := data.(map[string]any)
		require.True(t, ok, "Context value should be a map")

		// Data should be at root level
		assert.Equal(t, "value", contextMap["key"], "Key should be at root level")
	})
}

// TestAddDataToContextWithErrorHandling tests error propagation in the AddDataToContextHelper
func TestAddDataToContextWithErrorHandling(t *testing.T) {
	t.Parallel()

	// Create a test logger that discards output
	logger := slog.Default()

	t.Run("provider returns error and keeps original context", func(t *testing.T) {
		// Create a context provider
		provider := NewContextProvider(constants.EvalData)
		baseCtx := t.Context()

		// Add a mix of valid and invalid data to trigger an error
		enrichedCtx, err := AddDataToContextHelper(baseCtx, logger, provider,
			map[string]any{"valid": "data"},
			map[string]any{"": "empty-key"}, // Empty key will trigger an error
		)

		// Should return an error
		assert.Error(t, err)

		// Context should remain unchanged when there's an error
		assert.Equal(t, baseCtx, enrichedCtx, "Context should be unchanged when there's an error")

		// No data should be added to context
		data := enrichedCtx.Value(constants.EvalData)
		assert.Nil(t, data, "Context should not have data added when there's an error")
	})
}
