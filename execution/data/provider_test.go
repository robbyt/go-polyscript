package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/robbyt/go-polyscript/execution/constants"
)

// Mock provider that always returns an error
type mockErrorProvider struct{}

func (m *mockErrorProvider) GetData(ctx context.Context) (map[string]any, error) {
	return nil, assert.AnError
}

func (m *mockErrorProvider) AddDataToContext(ctx context.Context, data ...any) (context.Context, error) {
	return ctx, assert.AnError
}

func TestContextProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		contextKey      constants.ContextKey
		contextValue    any
		expectedSuccess bool
		expectedEmpty   bool
	}{
		{
			name:            "empty context key",
			contextKey:      "",
			contextValue:    nil,
			expectedSuccess: false,
			expectedEmpty:   true,
		},
		{
			name:            "nil context value",
			contextKey:      "test_key",
			contextValue:    nil,
			expectedSuccess: true,
			expectedEmpty:   true,
		},
		{
			name:       "valid data",
			contextKey: "test_key",
			contextValue: map[string]any{
				"foo": "bar",
				"baz": 123,
			},
			expectedSuccess: true,
			expectedEmpty:   false,
		},
		{
			name:            "wrong value type (string)",
			contextKey:      "test_key",
			contextValue:    "not a map",
			expectedSuccess: false,
			expectedEmpty:   true,
		},
		{
			name:            "wrong value type (int)",
			contextKey:      "test_key",
			contextValue:    42,
			expectedSuccess: false,
			expectedEmpty:   true,
		},
		{
			name:            "wrong map type",
			contextKey:      "test_key",
			contextValue:    map[int]string{1: "value"},
			expectedSuccess: false,
			expectedEmpty:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := NewContextProvider(tt.contextKey)
			require.NotNil(t, provider)

			ctx := context.Background()
			if tt.contextValue != nil {
				ctx = context.WithValue(ctx, tt.contextKey, tt.contextValue)
			}

			result, err := provider.GetData(ctx)

			if tt.expectedSuccess {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			if tt.expectedEmpty {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
				if validMap, ok := tt.contextValue.(map[string]any); ok {
					assert.Equal(t, validMap, result)
				}
			}
		})
	}
}

func TestStaticProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		staticData  map[string]any
		expectEmpty bool
	}{
		{
			name:        "nil data",
			staticData:  nil,
			expectEmpty: true,
		},
		{
			name:        "empty data",
			staticData:  map[string]any{},
			expectEmpty: true,
		},
		{
			name: "simple data",
			staticData: map[string]any{
				"foo": "bar",
				"baz": 123,
			},
			expectEmpty: false,
		},
		{
			name: "complex data",
			staticData: map[string]any{
				"string": "value",
				"int":    42,
				"bool":   true,
				"nested": map[string]any{
					"key": "nested value",
				},
				"array": []string{"one", "two"},
			},
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := NewStaticProvider(tt.staticData)
			require.NotNil(t, provider)

			// Context is not used by StaticProvider, so we can pass an empty one
			ctx := context.Background()
			result, err := provider.GetData(ctx)

			assert.NoError(t, err)

			if tt.expectEmpty {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
				assert.Equal(t, tt.staticData, result)

				// Test that we get a copy, not the original map
				originalLength := len(result)
				result["newKey"] = "newValue"

				newResult, _ := provider.GetData(ctx)
				assert.Len(t, newResult, originalLength)
				assert.NotContains(t, newResult, "newKey")
			}
		})
	}
}

func TestCompositeProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		providers      []Provider
		expectedKeys   []string
		expectedValues map[string]any
	}{
		{
			name:           "empty providers",
			providers:      []Provider{},
			expectedKeys:   []string{},
			expectedValues: map[string]any{},
		},
		{
			name: "single provider",
			providers: []Provider{
				NewStaticProvider(map[string]any{
					"key1": "value1",
					"key2": 2,
				}),
			},
			expectedKeys: []string{"key1", "key2"},
			expectedValues: map[string]any{
				"key1": "value1",
				"key2": 2,
			},
		},
		{
			name: "multiple providers with unique keys",
			providers: []Provider{
				NewStaticProvider(map[string]any{
					"key1": "value1",
					"key2": 2,
				}),
				NewStaticProvider(map[string]any{
					"key3": "value3",
					"key4": 4,
				}),
			},
			expectedKeys: []string{"key1", "key2", "key3", "key4"},
			expectedValues: map[string]any{
				"key1": "value1",
				"key2": 2,
				"key3": "value3",
				"key4": 4,
			},
		},
		{
			name: "multiple providers with overlapping keys (last one wins)",
			providers: []Provider{
				NewStaticProvider(map[string]any{
					"key1": "original1",
					"key2": "original2",
				}),
				NewStaticProvider(map[string]any{
					"key2": "override2",
					"key3": "value3",
				}),
			},
			expectedKeys: []string{"key1", "key2", "key3"},
			expectedValues: map[string]any{
				"key1": "original1",
				"key2": "override2", // This value should be from the second provider
				"key3": "value3",
			},
		},
		{
			name: "provider with error (should stop merging)",
			providers: []Provider{
				NewStaticProvider(map[string]any{
					"key1": "value1",
				}),
				// Create a mock provider that returns an error
				&mockErrorProvider{},
				NewStaticProvider(map[string]any{
					"key3": "value3", // This should not be merged
				}),
			},
			expectedKeys:   nil, // The error should prevent any merging
			expectedValues: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := NewCompositeProvider(tt.providers...)
			require.NotNil(t, provider)

			ctx := context.Background()
			result, err := provider.GetData(ctx)

			if tt.expectedValues == nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Check that all expected keys are present
				for _, key := range tt.expectedKeys {
					assert.Contains(t, result, key)
				}

				// Check that all values match expected values
				for key, expectedValue := range tt.expectedValues {
					assert.Equal(t, expectedValue, result[key])
				}
			}
		})
	}
}
