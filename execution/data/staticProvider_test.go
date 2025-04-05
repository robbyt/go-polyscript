package data

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStaticProvider_Creation tests the creation of StaticProvider instances
func TestStaticProvider_Creation(t *testing.T) {
	t.Parallel()

	t.Run("nil data creates empty map", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(nil)

		ctx := context.Background()
		result, err := provider.GetData(ctx)

		assert.NoError(t, err, "GetData should never return an error")
		assert.Empty(t, result, "Result map should be empty")
	})

	t.Run("empty data creates empty map", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(map[string]any{})

		ctx := context.Background()
		result, err := provider.GetData(ctx)

		assert.NoError(t, err, "GetData should never return an error")
		assert.Empty(t, result, "Result map should be empty")
	})

	t.Run("populated data is stored", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(simpleData)

		ctx := context.Background()
		result, err := provider.GetData(ctx)

		assert.NoError(t, err, "GetData should never return an error")
		assert.Equal(t, simpleData, result, "Result should match input data")
	})

	t.Run("complex data is stored", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(complexData)

		ctx := context.Background()
		result, err := provider.GetData(ctx)

		assert.NoError(t, err, "GetData should never return an error")
		assert.Equal(t, complexData, result, "Result should match input data")
	})
}

// TestStaticProvider_GetData tests the data retrieval functionality of StaticProvider
func TestStaticProvider_GetData(t *testing.T) {
	t.Parallel()

	t.Run("empty provider returns empty map", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(map[string]any{})
		ctx := context.Background()

		result, err := provider.GetData(ctx)

		assert.NoError(t, err, "GetData should never return an error")
		assert.Empty(t, result, "Result map should be empty")
	})

	t.Run("simple data", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(simpleData)
		ctx := context.Background()

		result, err := provider.GetData(ctx)

		assert.NoError(t, err, "GetData should never return an error")
		assert.Equal(t, simpleData, result, "Result should match input data")

		// Test that we get a copy, not the original map
		result["newTestKey"] = "newTestValue"

		newResult, err := provider.GetData(ctx)
		assert.NoError(t, err, "GetData should never return an error")
		assert.NotContains(
			t,
			newResult,
			"newTestKey",
			"Modifications to result should not affect provider",
		)
	})

	t.Run("complex nested data", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(complexData)
		ctx := context.Background()

		result, err := provider.GetData(ctx)

		assert.NoError(t, err, "GetData should never return an error")
		assert.Equal(t, complexData, result, "Result should match input data")
	})

	t.Run("nil provider data", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(nil)
		ctx := context.Background()

		result, err := provider.GetData(ctx)

		assert.NoError(t, err, "GetData should never return an error")
		assert.Empty(t, result, "Result map should be empty")
	})
}

// TestStaticProvider_AddDataToContext tests that StaticProvider properly rejects all context updates
func TestStaticProvider_AddDataToContext(t *testing.T) {
	t.Parallel()

	t.Run("nil context arg returns error", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(simpleData)
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx, nil)

		assert.Error(t, err, "StaticProvider should reject all attempts to add data")
		assert.True(t, errors.Is(err, ErrStaticProviderNoRuntimeUpdates),
			"Error should be ErrStaticProviderNoRuntimeUpdates")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")

		// Verify data is still available
		data, getErr := provider.GetData(ctx)
		assert.NoError(t, getErr)
		assert.Equal(t, simpleData, data)
	})

	t.Run("map context arg returns error", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(simpleData)
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"new": "data"})

		assert.Error(t, err, "StaticProvider should reject all attempts to add data")
		assert.True(t, errors.Is(err, ErrStaticProviderNoRuntimeUpdates),
			"Error should be ErrStaticProviderNoRuntimeUpdates")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
	})

	t.Run("HTTP request context arg returns error", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(simpleData)
		ctx := context.Background()
		req := createTestRequest()

		newCtx, err := provider.AddDataToContext(ctx, req)

		assert.Error(t, err, "StaticProvider should reject all attempts to add data")
		assert.True(t, errors.Is(err, ErrStaticProviderNoRuntimeUpdates),
			"Error should be ErrStaticProviderNoRuntimeUpdates")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
	})

	t.Run("multiple args returns error", func(t *testing.T) {
		t.Parallel()
		provider := NewStaticProvider(simpleData)
		ctx := context.Background()

		newCtx, err := provider.AddDataToContext(ctx,
			map[string]any{"key": "value"}, "string", 42)

		assert.Error(t, err, "StaticProvider should reject all attempts to add data")
		assert.True(t, errors.Is(err, ErrStaticProviderNoRuntimeUpdates),
			"Error should be ErrStaticProviderNoRuntimeUpdates")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
	})
}

// TestStaticProvider_ErrorIdentification tests error handling specifics for StaticProvider
func TestStaticProvider_ErrorIdentification(t *testing.T) {
	t.Parallel()

	provider := NewStaticProvider(simpleData)
	ctx := context.Background()

	_, err := provider.AddDataToContext(ctx, "some data")

	// Test that errors.Is works correctly with the sentinel error
	assert.True(t, errors.Is(err, ErrStaticProviderNoRuntimeUpdates),
		"Error should be identifiable with errors.Is")

	// Test direct equality for legacy code
	assert.Equal(t, ErrStaticProviderNoRuntimeUpdates, err,
		"Error should be the sentinel error directly")

	// Test error message content
	assert.Contains(t, err.Error(), "doesn't support adding data",
		"Error message should explain the limitation")

	// Verify data is still available
	data, getErr := provider.GetData(ctx)
	assert.NoError(t, getErr, "GetData should never return an error")
	assert.Equal(t, simpleData, data, "Static data should be available after error")
}
