package data

import (
	"context"
	"log/slog"
	"testing"

	"github.com/robbyt/go-polyscript/abstract/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrepareContextHelper tests the PrepareContextHelper utility function
func TestPrepareContextHelper(t *testing.T) {
	t.Parallel()

	// Create a test logger that discards output
	logger := slog.Default()

	t.Run("nil provider returns error", func(t *testing.T) {
		baseCtx := context.Background()
		enrichedCtx, err := PrepareContextHelper(
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
		baseCtx := context.Background()

		enrichedCtx, err := PrepareContextHelper(
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
		baseCtx := context.Background()

		enrichedCtx, err := PrepareContextHelper(
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

		// Data should be under input_data key
		assert.Contains(t, contextMap, constants.InputData)
		inputData, ok := contextMap[constants.InputData].(map[string]any)
		require.True(t, ok, "Input data should be a map")
		assert.Equal(t, "value", inputData["key"])
	})

	t.Run("context provider with HTTP request", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		baseCtx := context.Background()
		req := createTestRequestHelper()

		enrichedCtx, err := PrepareContextHelper(baseCtx, logger, provider, req)

		assert.NoError(t, err)
		assert.NotEqual(t, baseCtx, enrichedCtx, "Context should be modified")

		// Verify request data was added to context
		data := enrichedCtx.Value(constants.EvalData)
		require.NotNil(t, data)

		contextMap, ok := data.(map[string]any)
		require.True(t, ok, "Context value should be a map")

		// Data should be under request key
		assert.Contains(t, contextMap, constants.Request)
		requestData, ok := contextMap[constants.Request].(map[string]any)
		require.True(t, ok, "Request data should be a map")
		assert.Equal(t, "GET", requestData["Method"])
		assert.Equal(t, "/test", requestData["URL_Path"])
	})

	t.Run("context provider with mixed data", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		baseCtx := context.Background()
		req := createTestRequestHelper()

		enrichedCtx, err := PrepareContextHelper(baseCtx, logger, provider,
			map[string]any{"key": "value"}, req)

		assert.NoError(t, err)
		assert.NotEqual(t, baseCtx, enrichedCtx, "Context should be modified")

		// Verify data was added to context
		data := enrichedCtx.Value(constants.EvalData)
		require.NotNil(t, data)

		contextMap, ok := data.(map[string]any)
		require.True(t, ok, "Context value should be a map")

		// Verify input data
		assert.Contains(t, contextMap, constants.InputData)
		inputData, ok := contextMap[constants.InputData].(map[string]any)
		require.True(t, ok, "Input data should be a map")
		assert.Equal(t, "value", inputData["key"])

		// Verify request data
		assert.Contains(t, contextMap, constants.Request)
		requestData, ok := contextMap[constants.Request].(map[string]any)
		require.True(t, ok, "Request data should be a map")
		assert.Equal(t, "GET", requestData["Method"])
		assert.Equal(t, "/test", requestData["URL_Path"])
	})

	t.Run("context provider with unsupported data", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		baseCtx := context.Background()

		enrichedCtx, err := PrepareContextHelper(
			baseCtx,
			logger,
			provider,
			42,
		) // Integer is not supported

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
		baseCtx := context.Background()

		enrichedCtx, err := PrepareContextHelper(baseCtx, logger, provider,
			map[string]any{"key": "value"})

		assert.NoError(t, err)
		assert.NotEqual(t, baseCtx, enrichedCtx, "Context should be modified")

		// Verify data was added to context
		data := enrichedCtx.Value(constants.EvalData)
		require.NotNil(t, data)

		contextMap, ok := data.(map[string]any)
		require.True(t, ok, "Context value should be a map")

		// Data should be under input_data key
		assert.Contains(t, contextMap, constants.InputData)
	})
}

// TestPrepareContextWithErrorHandling tests error propagation in the PrepareContextHelper
func TestPrepareContextWithErrorHandling(t *testing.T) {
	t.Parallel()

	// Create a test logger that discards output
	logger := slog.Default()

	t.Run("provider returns error and keeps original context", func(t *testing.T) {
		// Create a context provider
		provider := NewContextProvider(constants.EvalData)
		baseCtx := context.Background()

		// Add a mix of valid and invalid data to trigger an error
		enrichedCtx, err := PrepareContextHelper(baseCtx, logger, provider,
			map[string]any{"valid": "data"},
			42, // Will trigger an error
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
