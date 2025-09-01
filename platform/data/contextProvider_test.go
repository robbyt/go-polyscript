package data

import (
	"context"
	"net/http"
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
	})

	t.Run("custom context key", func(t *testing.T) {
		provider := NewContextProvider("custom_key")

		assert.Equal(t, constants.ContextKey("custom_key"), provider.contextKey,
			"Context key should be set correctly")
	})

	t.Run("empty context key", func(t *testing.T) {
		provider := NewContextProvider("")

		assert.Equal(t, constants.ContextKey(""), provider.contextKey,
			"Context key should be set correctly")
	})
}

// TestContextProvider_GetData tests retrieving data from the context
func TestContextProvider_GetData(t *testing.T) {
	t.Parallel()

	t.Run("empty context key", func(t *testing.T) {
		provider := NewContextProvider("")
		ctx := t.Context()

		result, err := provider.GetData(ctx)

		assert.Error(t, err, "Should return error for empty context key")
		assert.Nil(t, result, "Result should be nil when error occurs")
	})

	t.Run("nil context value", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := t.Context()

		result, err := provider.GetData(ctx)

		assert.NoError(t, err, "Should not return error for nil context value")
		assert.NotNil(t, result, "Result should be an empty map, not nil")
		assert.Empty(t, result, "Result map should be empty")

		// Verify data consistency
		getDataCheckHelper(t, provider, ctx)
	})

	t.Run("valid simple data without namespace", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.WithValue(t.Context(), constants.EvalData, simpleData)

		result, err := provider.GetData(ctx)

		assert.NoError(t, err, "Should not return error for valid context")
		assert.Equal(t, simpleData, result, "Result should match expected data")

		// Verify data consistency
		getDataCheckHelper(t, provider, ctx)
	})

	t.Run("valid complex data without namespace", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.WithValue(t.Context(), constants.EvalData, complexData)

		result, err := provider.GetData(ctx)

		assert.NoError(t, err, "Should not return error for valid context")
		assert.Equal(t, complexData, result, "Result should match expected data")

		// Verify data consistency
		getDataCheckHelper(t, provider, ctx)
	})

	t.Run("nested data structures", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)

		// Create context with nested data
		data := map[string]any{
			"user": map[string]any{
				"profile": map[string]any{
					"name": "Alice",
					"age":  30,
				},
			},
			"settings": map[string]any{
				"theme": "dark",
			},
		}
		ctx := context.WithValue(t.Context(), constants.EvalData, data)

		result, err := provider.GetData(ctx)
		assert.NoError(t, err, "Should not return error for valid context")

		// Verify top-level keys exist
		assert.Contains(t, result, "user", "Should contain user key")
		assert.Contains(t, result, "settings", "Should contain settings key")

		// Verify nested data
		userMap, ok := result["user"].(map[string]any)
		assert.True(t, ok, "User should be a map")
		profileMap, ok := userMap["profile"].(map[string]any)
		assert.True(t, ok, "Profile should be a map")

		assert.Equal(t, "Alice", profileMap["name"], "Should have correct name")
		assert.Equal(t, 30, profileMap["age"], "Should have correct age")

		settingsMap, ok := result["settings"].(map[string]any)
		assert.True(t, ok, "Settings should be a map")
		assert.Equal(t, "dark", settingsMap["theme"], "Should have correct theme")
	})

	t.Run("mixed data types", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)

		// Create context with various data types
		data := map[string]any{
			"string": "value",
			"number": 42,
			"bool":   true,
			"array":  []string{"one", "two"},
			"map":    map[string]any{"key": "value"},
		}
		ctx := context.WithValue(t.Context(), constants.EvalData, data)

		result, err := provider.GetData(ctx)
		assert.NoError(t, err, "Should not return error for valid context")

		// Verify all types are preserved
		assert.Equal(t, "value", result["string"], "String should match")
		assert.Equal(t, 42, result["number"], "Number should match")
		assert.Equal(t, true, result["bool"], "Boolean should match")

		// Note that we can't assert directly on slices, but we can check length and contents
		arr, ok := result["array"].([]string)
		assert.True(t, ok, "Array should be preserved")
		assert.Equal(t, 2, len(arr), "Array should have correct length")

		nestedMap, ok := result["map"].(map[string]any)
		assert.True(t, ok, "Nested map should be preserved")
		assert.Equal(t, "value", nestedMap["key"], "Nested map should have correct values")
	})

	t.Run("invalid data type (string)", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.WithValue(t.Context(), constants.EvalData, "not a map")

		result, err := provider.GetData(ctx)

		assert.Error(t, err, "Should return error for invalid data type")
		assert.Nil(t, result, "Result should be nil when error occurs")
	})

	t.Run("invalid data type (int)", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.WithValue(t.Context(), constants.EvalData, 42)

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
		ctx := t.Context()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key": "value"})

		assert.Error(t, err, "Should return error for empty context key")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
	})

	t.Run("nil input data", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := t.Context()

		newCtx, err := provider.AddDataToContext(ctx, nil)

		assert.NoError(t, err, "Should not return error with nil data")
		assert.NotEqual(t, ctx, newCtx, "Context should be modified even with nil data")

		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)
		assert.Empty(t, data, "Data should be empty with nil input")
	})

	t.Run("simple map data", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := t.Context()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key1": "value1", "key2": 123})

		assert.NoError(t, err, "Should not return error with valid map data")
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)

		// Data should be at root level
		assert.Equal(t, "value1", data["key1"], "Should contain key1 at root level")
		assert.Equal(t, 123, data["key2"], "Should contain key2 at root level")
	})

	t.Run("multiple map data items", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := t.Context()

		newCtx, err := provider.AddDataToContext(ctx,
			map[string]any{"key1": "value1"},
			map[string]any{"key2": "value2"})

		assert.NoError(t, err, "Should not return error with multiple map items")
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)

		// Data should be at root level
		assert.Equal(t, "value1", data["key1"], "Should contain key1 at root level")
		assert.Equal(t, "value2", data["key2"], "Should contain key2 at root level")
	})

	t.Run("HTTP request as map value", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := t.Context()

		// Now we pass the request as a map value
		req := createTestRequestHelper()
		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"request": req})

		assert.NoError(t, err, "Should not return error with HTTP request in map")
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)
		assert.Contains(t, data, "request", "Should contain request key")

		requestData, ok := data["request"].(map[string]any)
		assert.True(t, ok, "request should be a map")
		assert.Equal(t, "GET", requestData["Method"], "Should contain HTTP method")
		assert.Equal(t, "/test", requestData["URL_Path"], "Should contain request path")
	})

	t.Run("empty string keys are rejected", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := t.Context()

		// Try to add data with an empty key
		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"": "value"})

		assert.Error(t, err, "Should error with empty key")
		assert.Contains(t, err.Error(), "empty keys are not allowed")

		// Context should be modified but the empty key should not be added
		data, getErr := provider.GetData(newCtx)
		assert.NoError(t, getErr)
		assert.NotContains(t, data, "", "Empty key should not be added")
	})

	t.Run("nested maps are processed recursively", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := t.Context()

		// Add nested data
		newCtx, err := provider.AddDataToContext(ctx, map[string]any{
			"user": map[string]any{
				"profile": map[string]any{
					"name": "Alice",
					"settings": map[string]any{
						"notifications": true,
					},
				},
			},
		})

		assert.NoError(t, err)

		data, getErr := provider.GetData(newCtx)
		assert.NoError(t, getErr)

		// Navigate the nested structure
		userMap, ok := data["user"].(map[string]any)
		assert.True(t, ok, "user should be a map")

		profileMap, ok := userMap["profile"].(map[string]any)
		assert.True(t, ok, "profile should be a map")

		assert.Equal(t, "Alice", profileMap["name"], "Should contain correct name")

		settingsMap, ok := profileMap["settings"].(map[string]any)
		assert.True(t, ok, "settings should be a map")
		assert.Equal(t, true, settingsMap["notifications"], "Should contain correct settings")
	})
}

// TestContextProvider_ProcessValue tests the processValue function directly
func TestContextProvider_ProcessValue(t *testing.T) {
	t.Parallel()

	t.Run("nil value", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		result, err := provider.processValue(nil)

		assert.NoError(t, err, "Should not error for nil value")
		assert.Nil(t, result, "Result should be nil")
	})

	t.Run("primitive types", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)

		// Test string
		result, err := provider.processValue("test string")
		assert.NoError(t, err)
		assert.Equal(t, "test string", result)

		// Test number
		result, err = provider.processValue(42)
		assert.NoError(t, err)
		assert.Equal(t, 42, result)

		// Test boolean
		result, err = provider.processValue(true)
		assert.NoError(t, err)
		assert.Equal(t, true, result)

		// Test slice
		slice := []string{"one", "two"}
		result, err = provider.processValue(slice)
		assert.NoError(t, err)
		assert.Equal(t, slice, result)
	})

	t.Run("nil http request pointer", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		var nilReq *http.Request = nil

		result, err := provider.processValue(nilReq)
		assert.NoError(t, err, "Should not error for nil HTTP request")
		assert.Nil(t, result, "Result should be nil")
	})

	t.Run("http request", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		req := createTestRequestHelper()

		// Test *http.Request
		result, err := provider.processValue(req)
		assert.NoError(t, err)
		resultMap, ok := result.(map[string]any)
		assert.True(t, ok, "Result should be a map")
		assert.Equal(t, "GET", resultMap["Method"])

		// Test http.Request (value)
		result, err = provider.processValue(*req)
		assert.NoError(t, err)
		resultMap, ok = result.(map[string]any)
		assert.True(t, ok, "Result should be a map")
		assert.Equal(t, "GET", resultMap["Method"])
	})

	t.Run("map with empty key", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		mapWithEmptyKey := map[string]any{
			"":      "value for empty key",
			"valid": "value",
		}

		_, err := provider.processValue(mapWithEmptyKey)
		assert.Error(t, err, "Should reject maps with empty keys")
		assert.Contains(t, err.Error(), "empty keys are not allowed")
	})

	t.Run("deeply nested map", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		nestedMap := map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": map[string]any{
						"value": 42,
					},
				},
			},
		}

		result, err := provider.processValue(nestedMap)
		assert.NoError(t, err)

		// Navigate through the levels to verify all maps were processed
		resultMap, ok := result.(map[string]any)
		assert.True(t, ok)

		level1, ok := resultMap["level1"].(map[string]any)
		assert.True(t, ok)

		level2, ok := level1["level2"].(map[string]any)
		assert.True(t, ok)

		level3, ok := level2["level3"].(map[string]any)
		assert.True(t, ok)

		assert.Equal(t, 42, level3["value"])
	})

	t.Run("nested map with error in deeper level", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		problematicMap := map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"": "this key is empty and should cause an error",
				},
			},
		}

		_, err := provider.processValue(problematicMap)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty keys are not allowed")
	})
}

// TestContextProvider_DataIntegration tests more complex data scenarios
func TestContextProvider_DataIntegration(t *testing.T) {
	t.Parallel()

	t.Run("basic map data", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := t.Context()

		// Add some data
		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key": "value"})
		require.NoError(t, err)

		// Verify data
		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)

		// Data should be available directly at root level
		assert.Equal(t, "value", data["key"], "Should contain the correct value at root")
	})

	t.Run("should preserve context data across calls", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)

		// Create a context directly with data already in it
		existingData := map[string]any{
			"existing": "value",
		}
		ctx := context.WithValue(t.Context(), constants.EvalData, existingData)

		// Add more data
		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"new": "value"})
		require.NoError(t, err)

		// Verify both pieces of data exist
		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)

		assert.Equal(t, "value", data["existing"], "Should preserve existing value")
		assert.Equal(t, "value", data["new"], "Should add new value")
	})

	t.Run("HTTP request as map value", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := t.Context()

		// Create request
		req := createTestRequestHelper()

		// Add an HTTP request as a value in a map
		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"request": req})
		require.NoError(t, err)

		// Get raw data from context
		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)

		// Request should be converted to a map
		requestData, ok := data["request"].(map[string]any)
		assert.True(t, ok, "Request should be a map")
		assert.Equal(t, "GET", requestData["Method"], "Should contain HTTP method")
	})

	t.Run("should recursively merge nested maps", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := t.Context()

		// Add initial nested data
		newCtx, err := provider.AddDataToContext(ctx, map[string]any{
			"user": map[string]any{
				"name": "Alice",
				"settings": map[string]any{
					"theme": "dark",
				},
			},
		})
		require.NoError(t, err)

		// Add more data that should be merged with existing
		newCtx, err = provider.AddDataToContext(newCtx, map[string]any{
			"user": map[string]any{
				"age": 30,
				"settings": map[string]any{
					"notifications": true,
				},
			},
		})
		require.NoError(t, err)

		// Verify all data correctly merged
		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)

		userMap, ok := data["user"].(map[string]any)
		assert.True(t, ok, "User should be a map")

		assert.Equal(t, "Alice", userMap["name"], "Should retain original name")
		assert.Equal(t, 30, userMap["age"], "Should add new age field")

		settings, ok := userMap["settings"].(map[string]any)
		assert.True(t, ok, "Settings should be a map")
		assert.Equal(t, "dark", settings["theme"], "Should retain original theme setting")
		assert.Equal(t, true, settings["notifications"], "Should add new notifications setting")
	})

	t.Run("multiple providers with different context keys", func(t *testing.T) {
		provider1 := NewContextProvider("user_data")
		provider2 := NewContextProvider("system_data")

		ctx := t.Context()

		// Add data with first provider
		ctx, err1 := provider1.AddDataToContext(ctx, map[string]any{"name": "Alice"})
		require.NoError(t, err1)

		// Add data with second provider
		ctx, err2 := provider2.AddDataToContext(ctx, map[string]any{"version": "1.0"})
		require.NoError(t, err2)

		// Get data from both providers
		userData, err3 := provider1.GetData(ctx)
		systemData, err4 := provider2.GetData(ctx)

		require.NoError(t, err3)
		require.NoError(t, err4)

		// Each provider should see its own data
		assert.Equal(t, "Alice", userData["name"], "User provider should see name")
		assert.Equal(t, "1.0", systemData["version"], "System provider should see version")
	})

	t.Run("should reject empty keys", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := t.Context()

		// Add data with an empty key
		badData := map[string]any{"": "value"}
		_, err := provider.AddDataToContext(ctx, badData)

		assert.Error(t, err, "Should reject empty keys")
		assert.Contains(t, err.Error(), "empty keys are not allowed")
	})
}
