package data

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStaticProvider_Creation tests the creation of StaticProvider instances
func TestStaticProvider_Creation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		inputData   map[string]any
		expectEmpty bool
	}{
		{
			name:        "nil data creates empty map",
			inputData:   nil,
			expectEmpty: true,
		},
		{
			name:        "empty data creates empty map",
			inputData:   map[string]any{},
			expectEmpty: true,
		},
		{
			name:        "populated data is stored",
			inputData:   simpleData,
			expectEmpty: false,
		},
		{
			name:        "complex data is stored",
			inputData:   complexData,
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewStaticProvider(tt.inputData)
			require.NotNil(t, provider, "Provider should never be nil")

			ctx := t.Context()
			result, err := provider.GetData(ctx)

			assert.NoError(t, err, "GetData should never return an error")

			if tt.expectEmpty {
				assert.Empty(t, result, "Result map should be empty")
			} else {
				assert.Equal(t, tt.inputData, result, "Result should match input data")
			}
		})
	}
}

// TestStaticProvider_GetData tests the data retrieval functionality of StaticProvider
func TestStaticProvider_GetData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputData    map[string]any
		modifyResult bool // Flag to check if modifying result affects provider's data
	}{
		{
			name:         "empty provider returns empty map",
			inputData:    map[string]any{},
			modifyResult: false,
		},
		{
			name:         "simple data",
			inputData:    simpleData,
			modifyResult: true,
		},
		{
			name:         "complex nested data",
			inputData:    complexData,
			modifyResult: true,
		},
		{
			name:         "nil provider data",
			inputData:    nil,
			modifyResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewStaticProvider(tt.inputData)
			ctx := t.Context()

			result, err := provider.GetData(ctx)

			assert.NoError(t, err, "GetData should never return an error")

			if tt.inputData == nil {
				assert.Empty(t, result, "Result map should be empty for nil input")
			} else {
				assert.Equal(t, tt.inputData, result, "Result should match input data")
			}

			// Verify data consistency and immutability
			if tt.modifyResult {
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
			}

			// Verify data consistency
			getDataCheckHelper(t, provider, ctx)
		})
	}
}

// TestStaticProvider_AddDataToContext tests that StaticProvider properly rejects all context updates
func TestStaticProvider_AddDataToContext(t *testing.T) {
	t.Parallel()

	t.Run("nil context arg returns error", func(t *testing.T) {
		provider := NewStaticProvider(simpleData)
		ctx := t.Context()

		newCtx, err := provider.AddDataToContext(ctx, nil)

		assert.Error(t, err, "StaticProvider should reject all attempts to add data")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
		assert.True(t, errors.Is(err, ErrStaticProviderNoRuntimeUpdates),
			"Error should be ErrStaticProviderNoRuntimeUpdates")

		// Verify data is still available
		data, getErr := provider.GetData(ctx)
		assert.NoError(t, getErr)
		assert.Equal(t, simpleData, data)
	})

	t.Run("map context arg returns error", func(t *testing.T) {
		provider := NewStaticProvider(simpleData)
		ctx := t.Context()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"new": "data"})

		assert.Error(t, err, "StaticProvider should reject all attempts to add data")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
		assert.True(t, errors.Is(err, ErrStaticProviderNoRuntimeUpdates),
			"Error should be ErrStaticProviderNoRuntimeUpdates")
	})

	t.Run("HTTP request context arg returns error", func(t *testing.T) {
		provider := NewStaticProvider(simpleData)
		ctx := t.Context()

		newCtx, err := provider.AddDataToContext(
			ctx,
			map[string]any{"request": createTestRequestHelper()},
		)

		assert.Error(t, err, "StaticProvider should reject all attempts to add data")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
		assert.True(t, errors.Is(err, ErrStaticProviderNoRuntimeUpdates),
			"Error should be ErrStaticProviderNoRuntimeUpdates")
	})

	t.Run("multiple args returns error", func(t *testing.T) {
		provider := NewStaticProvider(simpleData)
		ctx := t.Context()

		newCtx, err := provider.AddDataToContext(
			ctx,
			map[string]any{"key": "value"},
			map[string]any{"str": "string"},
			map[string]any{"num": 42},
		)

		assert.Error(t, err, "StaticProvider should reject all attempts to add data")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
		assert.True(t, errors.Is(err, ErrStaticProviderNoRuntimeUpdates),
			"Error should be ErrStaticProviderNoRuntimeUpdates")
	})
}

// TestStaticProvider_ErrorIdentification tests error handling specifics for StaticProvider
func TestStaticProvider_ErrorIdentification(t *testing.T) {
	t.Parallel()

	provider := NewStaticProvider(simpleData)
	ctx := t.Context()

	_, err := provider.AddDataToContext(ctx, map[string]any{"data": "some data"})

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
