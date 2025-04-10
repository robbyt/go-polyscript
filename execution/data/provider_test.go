package data

import (
	"context"
	"errors"
	"testing"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/stretchr/testify/assert"
)

// TestProvider_Interface ensures that all provider implementations comply with the Provider interface
func TestProvider_Interface(t *testing.T) {
	t.Parallel()

	t.Run("StaticProvider implements Provider", func(t *testing.T) {
		var _ Provider = &StaticProvider{}
	})

	t.Run("ContextProvider implements Provider", func(t *testing.T) {
		var _ Provider = &ContextProvider{}
	})

	t.Run("CompositeProvider implements Provider", func(t *testing.T) {
		var _ Provider = &CompositeProvider{}
	})

	t.Run("MockProvider implements Provider", func(t *testing.T) {
		var _ Provider = &MockProvider{}
	})
}

// TestProvider_GetData tests the basic GetData functionality across all provider types
func TestProvider_GetData(t *testing.T) {
	t.Parallel()

	// Test static provider
	t.Run("static provider with simple data", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(simpleData)
		ctx := context.Background()

		result, err := provider.GetData(ctx)
		assert.NoError(t, err)
		assert.Equal(t, simpleData, result)

		// Get a fresh copy to verify data consistency
		newResult, err := provider.GetData(ctx)
		assert.NoError(t, err)
		assert.Equal(t, simpleData, newResult)
	})

	t.Run("static provider with empty data", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(nil)
		ctx := context.Background()

		result, err := provider.GetData(ctx)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	// Test context provider
	t.Run("context provider with valid data", func(t *testing.T) {
		t.Parallel()
		provider := NewContextProvider(constants.EvalData)
		ctx := context.WithValue(context.Background(), constants.EvalData, simpleData)

		result, err := provider.GetData(ctx)
		assert.NoError(t, err)
		assert.Equal(t, simpleData, result)

		// Get a fresh copy to verify data consistency
		newResult, err := provider.GetData(ctx)
		assert.NoError(t, err)
		assert.Equal(t, simpleData, newResult)
	})

	t.Run("context provider with empty key", func(t *testing.T) {
		t.Parallel()
		provider := NewContextProvider("")
		ctx := context.Background()

		result, err := provider.GetData(ctx)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("context provider with invalid value type", func(t *testing.T) {
		t.Parallel()
		provider := NewContextProvider(constants.EvalData)
		ctx := context.WithValue(context.Background(), constants.EvalData, "not a map")

		result, err := provider.GetData(ctx)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	// Test composite provider
	t.Run("composite provider with multiple sources", func(t *testing.T) {
		t.Parallel()
		provider := NewCompositeProvider(
			NewStaticProvider(map[string]any{"static": "value", "shared": "static"}),
			NewContextProvider(constants.EvalData),
		)

		ctx := context.WithValue(
			context.Background(),
			constants.EvalData,
			map[string]any{"context": "value", "shared": "context"},
		)

		result, err := provider.GetData(ctx)
		assert.NoError(t, err)

		// Verify expected values (context overrides static for shared keys)
		assert.Equal(t, "value", result["static"])
		assert.Equal(t, "value", result["context"])
		assert.Equal(
			t,
			"context",
			result["shared"],
			"Context provider should override static provider",
		)

		// Get a fresh copy to verify data consistency
		newResult, err := provider.GetData(ctx)
		assert.NoError(t, err)
		assert.Equal(t, result, newResult)
	})

	t.Run("empty composite provider", func(t *testing.T) {
		t.Parallel()
		provider := NewCompositeProvider()
		ctx := context.Background()

		result, err := provider.GetData(ctx)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("composite provider with error", func(t *testing.T) {
		t.Parallel()
		provider := NewCompositeProvider(
			NewStaticProvider(simpleData),
			newMockErrorProvider(),
		)
		ctx := context.Background()

		result, err := provider.GetData(ctx)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestProvider_AddDataToContext tests adding data to context across provider implementations
func TestProvider_AddDataToContext(t *testing.T) {
	t.Parallel()

	// Test with static provider
	t.Run("static provider should reject all data", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(simpleData)
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key": "value"})

		assert.Error(t, err, "StaticProvider should reject data additions")
		assert.True(t, errors.Is(err, ErrStaticProviderNoRuntimeUpdates))
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")

		// Verify static data is still available
		data, err := provider.GetData(ctx)
		assert.NoError(t, err)
		assert.Equal(t, simpleData, data)
	})

	// Test with context provider
	t.Run("context provider with valid map data", func(t *testing.T) {
		t.Parallel()
		provider := NewContextProvider(constants.EvalData)
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key": "value"})

		assert.NoError(t, err)
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		// Verify data was stored correctly
		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)

		expectedData := map[string]any{
			constants.InputData: map[string]any{"key": "value"},
		}
		assert.Equal(t, expectedData, data)
	})

	t.Run("context provider with HTTP request", func(t *testing.T) {
		t.Parallel()
		provider := NewContextProvider(constants.EvalData)
		ctx := context.Background()
		req := createTestRequest()

		newCtx, err := provider.AddDataToContext(ctx, req)

		assert.NoError(t, err)
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		// Verify request data was stored correctly
		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)

		requestMap, ok := data[constants.Request].(map[string]any)
		assert.True(t, ok, "Request data should be a map")
		assert.Equal(t, "GET", requestMap["Method"])
		assert.Equal(t, "/test", requestMap["URL_Path"])
	})

	t.Run("context provider with empty key", func(t *testing.T) {
		t.Parallel()
		provider := NewContextProvider("")
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key": "value"})

		assert.Error(t, err, "Empty context key should cause error")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged on error")
	})

	// Test with composite provider
	t.Run("composite provider with mixed providers", func(t *testing.T) {
		t.Parallel()
		provider := NewCompositeProvider(
			NewStaticProvider(simpleData),
			NewContextProvider(constants.EvalData),
		)
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key": "value"})

		assert.NoError(
			t,
			err,
			"StaticProvider error should be ignored when ContextProvider succeeds",
		)
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		// Verify data
		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)

		// Should have both static and context data
		assert.Equal(t, simpleData["string"], data["string"], "Should contain static data")
		assert.Contains(t, data, constants.InputData, "Should contain input_data key")

		inputData, ok := data[constants.InputData].(map[string]any)
		assert.True(t, ok, "input_data should be a map")
		assert.Equal(t, "value", inputData["key"], "Should contain added data")
	})

	t.Run("composite provider with all failures", func(t *testing.T) {
		t.Parallel()
		provider := NewCompositeProvider(
			NewStaticProvider(simpleData),
			newMockErrorProvider(),
		)
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key": "value"})

		assert.Error(t, err, "Should error when all providers fail")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
	})

	// Test with multiple data items
	t.Run("context provider with multiple data items", func(t *testing.T) {
		t.Parallel()
		provider := NewContextProvider(constants.EvalData)
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx,
			map[string]any{"key1": "value1"},
			map[string]any{"key2": "value2"})

		assert.NoError(t, err)
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		// Verify data was merged correctly
		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)

		inputData, ok := data[constants.InputData].(map[string]any)
		assert.True(t, ok, "input_data should be a map")
		assert.Equal(t, "value1", inputData["key1"], "Should contain first item")
		assert.Equal(t, "value2", inputData["key2"], "Should contain second item")
	})
}

// TestProvider_DeepMerge tests the deep merge functionality specifically
func TestProvider_DeepMerge(t *testing.T) {
	t.Parallel()

	t.Run("simple merge with no overlaps", func(t *testing.T) {
		t.Parallel()
		src := map[string]any{"src_key": "src_value"}
		dst := map[string]any{"dst_key": "dst_value"}
		expected := map[string]any{
			"src_key": "src_value",
			"dst_key": "dst_value",
		}

		result := deepMerge(src, dst)
		assert.Equal(t, expected, result)

		// Ensure original maps weren't modified
		assert.Equal(t, map[string]any{"src_key": "src_value"}, src)
		assert.Equal(t, map[string]any{"dst_key": "dst_value"}, dst)
	})

	t.Run("overlapping keys (dst wins)", func(t *testing.T) {
		t.Parallel()
		src := map[string]any{
			"shared_key": "src_value",
			"src_key":    "src_value",
		}
		dst := map[string]any{
			"shared_key": "dst_value",
			"dst_key":    "dst_value",
		}
		expected := map[string]any{
			"shared_key": "dst_value", // dst wins
			"src_key":    "src_value",
			"dst_key":    "dst_value",
		}

		result := deepMerge(src, dst)
		assert.Equal(t, expected, result)
	})

	t.Run("nested maps are merged properly", func(t *testing.T) {
		t.Parallel()
		src := map[string]any{
			"nested": map[string]any{
				"key1": "src_value1",
				"key2": "src_value2",
			},
		}
		dst := map[string]any{
			"nested": map[string]any{
				"key2": "dst_value2", // Overrides src
				"key3": "dst_value3", // New key
			},
		}
		expected := map[string]any{
			"nested": map[string]any{
				"key1": "src_value1", // Preserved from src
				"key2": "dst_value2", // Overridden by dst
				"key3": "dst_value3", // Added from dst
			},
		}

		result := deepMerge(src, dst)
		assert.Equal(t, expected, result)
	})

	t.Run("arrays are replaced not merged", func(t *testing.T) {
		t.Parallel()
		src := map[string]any{"array": []string{"one", "two", "three"}}
		dst := map[string]any{"array": []string{"four", "five"}}
		expected := map[string]any{"array": []string{"four", "five"}}

		result := deepMerge(src, dst)
		assert.Equal(t, expected, result)
	})

	t.Run("empty maps", func(t *testing.T) {
		t.Parallel()
		result1 := deepMerge(map[string]any{}, map[string]any{"key": "value"})
		assert.Equal(t, map[string]any{"key": "value"}, result1)

		result2 := deepMerge(map[string]any{"key": "value"}, map[string]any{})
		assert.Equal(t, map[string]any{"key": "value"}, result2)
	})

	t.Run("original maps should not be modified", func(t *testing.T) {
		t.Parallel()
		src := map[string]any{
			"key": "value",
			"nested": map[string]any{
				"inner": "original",
			},
		}
		dst := map[string]any{
			"key": "new-value",
			"nested": map[string]any{
				"inner": "modified",
				"added": "new-field",
			},
		}

		srcCopy := map[string]any{
			"key": "value",
			"nested": map[string]any{
				"inner": "original",
			},
		}

		dstCopy := map[string]any{
			"key": "new-value",
			"nested": map[string]any{
				"inner": "modified",
				"added": "new-field",
			},
		}

		_ = deepMerge(src, dst)

		// Original maps should not be modified
		assert.Equal(t, srcCopy, src)
		assert.Equal(t, dstCopy, dst)
	})
}
