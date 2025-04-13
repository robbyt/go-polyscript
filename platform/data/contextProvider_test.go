package data

import (
	"context"
	"testing"

	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContextProvider_Creation tests the creation and initialization of ContextProvider
func TestContextProvider_Creation(t *testing.T) {
	t.Parallel()

	t.Run("standard context key", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)

		assert.Equal(t, constants.EvalData, provider.contextKey,
			"Context key should be set correctly")
		assert.Equal(t, constants.InputData, provider.storageKey,
			"Storage key should be initialized")
		assert.Equal(t, constants.Request, provider.requestKey,
			"Request key should be initialized")
		assert.Equal(t, constants.Response, provider.responseKey,
			"Response key should be initialized")
	})

	t.Run("custom context key", func(t *testing.T) {
		provider := NewContextProvider("custom_key")

		assert.Equal(t, constants.ContextKey("custom_key"), provider.contextKey,
			"Context key should be set correctly")
		assert.Equal(t, constants.InputData, provider.storageKey,
			"Storage key should be initialized")
	})

	t.Run("empty context key", func(t *testing.T) {
		provider := NewContextProvider("")

		assert.Equal(t, constants.ContextKey(""), provider.contextKey,
			"Context key should be set correctly")
		assert.Equal(t, constants.InputData, provider.storageKey,
			"Storage key should be initialized")
	})
}

// TestContextProvider_GetData tests retrieving data from the context
func TestContextProvider_GetData(t *testing.T) {
	t.Parallel()

	t.Run("empty context key", func(t *testing.T) {
		provider := NewContextProvider("")
		ctx := context.Background()

		result, err := provider.GetData(ctx)

		assert.Error(t, err, "Should return error for empty context key")
		assert.Nil(t, result, "Result should be nil when error occurs")
	})

	t.Run("nil context value", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.Background()

		result, err := provider.GetData(ctx)

		assert.NoError(t, err, "Should not return error for nil context value")
		assert.NotNil(t, result, "Result should be an empty map, not nil")
		assert.Empty(t, result, "Result map should be empty")

		// Verify data consistency
		getDataCheckHelper(t, provider, ctx)
	})

	t.Run("valid simple data", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.WithValue(context.Background(), constants.EvalData, simpleData)

		result, err := provider.GetData(ctx)

		assert.NoError(t, err, "Should not return error for valid context")
		assert.Equal(t, simpleData, result, "Result should match expected data")

		// Verify data consistency
		getDataCheckHelper(t, provider, ctx)
	})

	t.Run("valid complex data", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.WithValue(context.Background(), constants.EvalData, complexData)

		result, err := provider.GetData(ctx)

		assert.NoError(t, err, "Should not return error for valid context")
		assert.Equal(t, complexData, result, "Result should match expected data")

		// Verify data consistency
		getDataCheckHelper(t, provider, ctx)
	})

	t.Run("invalid data type (string)", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.WithValue(context.Background(), constants.EvalData, "not a map")

		result, err := provider.GetData(ctx)

		assert.Error(t, err, "Should return error for invalid data type")
		assert.Nil(t, result, "Result should be nil when error occurs")
	})

	t.Run("invalid data type (int)", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.WithValue(context.Background(), constants.EvalData, 42)

		result, err := provider.GetData(ctx)

		assert.Error(t, err, "Should return error for invalid data type")
		assert.Nil(t, result, "Result should be nil when error occurs")
	})
}

// TestContextProvider_AddDataToContext tests adding different types of data to context
func TestContextProvider_AddDataToContext(t *testing.T) {
	t.Parallel()

	t.Run("empty context key", func(t *testing.T) {
		provider := NewContextProvider("")
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key": "value"})

		assert.Error(t, err, "Should return error for empty context key")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
	})

	t.Run("nil input data", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx, nil)

		assert.NoError(t, err, "Should not return error with nil data")
		assert.NotEqual(t, ctx, newCtx, "Context should be modified even with nil data")

		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)
		assert.Empty(t, data, "Data should be empty with nil input")
	})

	t.Run("simple map data", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key1": "value1", "key2": 123})

		assert.NoError(t, err, "Should not return error with valid map data")
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)
		assert.Contains(t, data, constants.InputData, "Should contain input_data key")

		inputData, ok := data[constants.InputData].(map[string]any)
		assert.True(t, ok, "input_data should be a map")
		assert.Equal(t, "value1", inputData["key1"], "Should contain key1")
		assert.Equal(t, 123, inputData["key2"], "Should contain key2")
	})

	t.Run("multiple map data items", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx,
			map[string]any{"key1": "value1"},
			map[string]any{"key2": "value2"})

		assert.NoError(t, err, "Should not return error with multiple map items")
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)
		assert.Contains(t, data, constants.InputData, "Should contain input_data key")

		inputData, ok := data[constants.InputData].(map[string]any)
		assert.True(t, ok, "input_data should be a map")
		assert.Equal(t, "value1", inputData["key1"], "Should contain key1")
		assert.Equal(t, "value2", inputData["key2"], "Should contain key2")
	})

	t.Run("HTTP request data", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx, createTestRequestHelper())

		assert.NoError(t, err, "Should not return error with HTTP request")
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)
		assert.Contains(t, data, constants.Request, "Should contain request key")

		requestData, ok := data[constants.Request].(map[string]any)
		assert.True(t, ok, "request should be a map")
		assert.Equal(t, "GET", requestData["Method"], "Should contain HTTP method")
		assert.Equal(t, "/test", requestData["URL_Path"], "Should contain request path")
	})

	t.Run("unsupported data type", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx, 42) // Integer is not supported

		assert.Error(t, err, "Should error with unsupported data type")
		assert.NotEqual(t, ctx, newCtx, "Context should be modified despite error")

		// Context should be modified but empty
		data, getErr := provider.GetData(newCtx)
		assert.NoError(t, getErr, "GetData should work after AddDataToContext")
		assert.NotNil(t, data, "Data should not be nil despite error")
		assert.Empty(t, data, "Data should be empty with unsupported type")
	})

	t.Run("mixed supported and unsupported", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.Background()

		// Only the map is supported
		newCtx, err := provider.AddDataToContext(ctx,
			map[string]any{"key": "value"},
			42,
			"string")

		assert.Error(t, err, "Should error with unsupported data types")
		assert.NotEqual(t, ctx, newCtx, "Context should be modified despite error")

		data, getErr := provider.GetData(newCtx)
		assert.NoError(t, getErr, "GetData should work after AddDataToContext")
		assert.Contains(t, data, constants.InputData, "Should contain input_data key")

		inputData, ok := data[constants.InputData].(map[string]any)
		assert.True(t, ok, "input_data should be a map")
		assert.Equal(t, "value", inputData["key"], "Should contain supported data")
	})
}

// TestContextProvider_DataIntegration tests more complex data scenarios
func TestContextProvider_DataIntegration(t *testing.T) {
	t.Parallel()

	t.Run("single map data item", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.Background()

		// Add some data
		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key": "value"})
		require.NoError(t, err)

		// Verify data
		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)

		// Check for input_data key
		assert.Contains(t, data, constants.InputData, "Should contain input_data key")

		// Verify the input_data values
		inputData, ok := data[constants.InputData].(map[string]any)
		assert.True(t, ok, "input_data should be a map")
		assert.Equal(t, "value", inputData["key"], "Should contain the correct value")
	})

	t.Run("should preserve context data across calls", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)

		// Create a context directly with data already in it
		existingData := map[string]any{
			constants.InputData: map[string]any{"existing": "value"},
		}
		ctx := context.WithValue(context.Background(), constants.EvalData, existingData)

		// Add more data
		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"new": "value"})
		require.NoError(t, err)

		// Verify both pieces of data exist
		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)
		assert.Contains(t, data, constants.InputData, "Should contain input_data key")

		inputData, ok := data[constants.InputData].(map[string]any)
		assert.True(t, ok, "input_data should be a map")
		assert.Equal(t, "value", inputData["existing"], "Should preserve existing value")
		assert.Equal(t, "value", inputData["new"], "Should add new value")
	})
}
