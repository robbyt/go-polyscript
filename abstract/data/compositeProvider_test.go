package data

import (
	"context"
	"errors"
	"testing"

	"github.com/robbyt/go-polyscript/abstract/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompositeProvider_Creation tests the creation of CompositeProvider instances
func TestCompositeProvider_Creation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		providers      []Provider
		expectedLength int
	}{
		{
			name:           "empty providers",
			providers:      []Provider{},
			expectedLength: 0,
		},
		{
			name:           "single provider",
			providers:      []Provider{NewStaticProvider(simpleData)},
			expectedLength: 1,
		},
		{
			name: "multiple providers",
			providers: []Provider{
				NewStaticProvider(simpleData),
				NewContextProvider(constants.EvalData),
			},
			expectedLength: 2,
		},
		{
			name: "providers with nil",
			providers: []Provider{
				NewStaticProvider(simpleData),
				nil,
				NewContextProvider(constants.EvalData),
			},
			expectedLength: 3, // The nil provider is still stored, just skipped during operations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			composite := NewCompositeProvider(tt.providers...)
			require.NotNil(t, composite, "CompositeProvider should never be nil")
			assert.Len(
				t,
				composite.providers,
				tt.expectedLength,
				"Provider list length should match",
			)
		})
	}
}

// TestCompositeProvider_GetData tests the data retrieval functionality of CompositeProvider
func TestCompositeProvider_GetData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupProvider func() Provider
		setupContext  func() context.Context
		expectedData  map[string]any
		expectError   bool
	}{
		{
			name: "empty provider returns empty map",
			setupProvider: func() Provider {
				return NewCompositeProvider()
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedData: map[string]any{},
			expectError:  false,
		},
		{
			name: "single static provider",
			setupProvider: func() Provider {
				return NewCompositeProvider(NewStaticProvider(simpleData))
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedData: simpleData,
			expectError:  false,
		},
		{
			name: "single context provider",
			setupProvider: func() Provider {
				return NewCompositeProvider(NewContextProvider(constants.EvalData))
			},
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), constants.EvalData, simpleData)
			},
			expectedData: simpleData,
			expectError:  false,
		},
		{
			name: "static and context providers with no overlap",
			setupProvider: func() Provider {
				return NewCompositeProvider(
					NewStaticProvider(map[string]any{"static_key": "static_value"}),
					NewContextProvider(constants.EvalData),
				)
			},
			setupContext: func() context.Context {
				return context.WithValue(
					context.Background(),
					constants.EvalData,
					map[string]any{"runtime_key": "runtime_value"},
				)
			},
			expectedData: map[string]any{
				"static_key":  "static_value",
				"runtime_key": "runtime_value",
			},
			expectError: false,
		},
		{
			name: "static and context providers with overlap (context wins)",
			setupProvider: func() Provider {
				return NewCompositeProvider(
					NewStaticProvider(map[string]any{
						"shared_key": "static_value",
						"static_key": "static_value",
					}),
					NewContextProvider(constants.EvalData),
				)
			},
			setupContext: func() context.Context {
				return context.WithValue(
					context.Background(),
					constants.EvalData,
					map[string]any{
						"shared_key":  "runtime_value",
						"runtime_key": "runtime_value",
					})
			},
			expectedData: map[string]any{
				"shared_key":  "runtime_value", // Context provider overrides static provider
				"static_key":  "static_value",
				"runtime_key": "runtime_value",
			},
			expectError: false,
		},
		{
			name: "nested data structures merge properly",
			setupProvider: func() Provider {
				return NewCompositeProvider(
					NewStaticProvider(map[string]any{
						"config": map[string]any{
							"timeout": 30,
							"retries": 3,
						},
					}),
					NewContextProvider(constants.EvalData),
				)
			},
			setupContext: func() context.Context {
				data := map[string]any{
					"input": "API User",
					"request": map[string]any{
						"id": "123",
					},
					"config": map[string]any{
						"host":    "localhost:8080", // New key in existing map
						"retries": 5,                // Override existing key
					},
				}
				return context.WithValue(context.Background(), constants.EvalData, data)
			},
			expectedData: map[string]any{
				"config": map[string]any{
					"timeout": 30,
					"retries": 5,                // Overridden
					"host":    "localhost:8080", // Added
				},
				"input": "API User",
				"request": map[string]any{
					"id": "123",
				},
			},
			expectError: false,
		},
		{
			name: "provider with error stops data retrieval",
			setupProvider: func() Provider {
				return NewCompositeProvider(
					NewStaticProvider(simpleData),
					newMockErrorProvider(),
				)
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedData: nil,
			expectError:  true,
		},
		{
			name: "nil providers are skipped",
			setupProvider: func() Provider {
				return NewCompositeProvider(
					NewStaticProvider(map[string]any{"key1": "value1"}),
					nil,
					NewStaticProvider(map[string]any{"key2": "value2"}),
				)
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedData: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := tt.setupProvider()
			require.NotNil(t, provider, "Provider should never be nil")

			ctx := tt.setupContext()
			result, err := provider.GetData(ctx)

			if tt.expectError {
				assert.Error(t, err, "Should return error when a provider fails")
				return
			}

			assert.NoError(t, err, "Should not return error for valid providers")
			assertMapContainsExpectedHelper(t, tt.expectedData, result)

			// Verify data consistency across calls
			getDataCheckHelper(t, provider, ctx)
		})
	}
}

// TestCompositeProvider_AddDataToContext tests adding data to context via the composite provider
func TestCompositeProvider_AddDataToContext(t *testing.T) {
	t.Parallel()

	t.Run("empty providers list", func(t *testing.T) {
		provider := NewCompositeProvider()
		require.NotNil(t, provider)

		ctx := context.Background()
		inputData := map[string]any{"key": "value"}

		newCtx, err := provider.AddDataToContext(ctx, inputData)

		assert.NoError(t, err, "Should not return error with empty provider list")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
	})

	t.Run("single context provider succeeds", func(t *testing.T) {
		provider := NewCompositeProvider(NewContextProvider(constants.EvalData))
		require.NotNil(t, provider)

		ctx := context.Background()
		inputData := map[string]any{"key": "value"}

		newCtx, err := provider.AddDataToContext(ctx, inputData)

		assert.NoError(t, err, "Should not return error for context provider")
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		// Verify data was added correctly
		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)
		assert.Contains(t, data, constants.InputData)

		inputDataResult, ok := data[constants.InputData].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "value", inputDataResult["key"])
	})

	t.Run("single static provider always errors", func(t *testing.T) {
		provider := NewCompositeProvider(NewStaticProvider(simpleData))
		require.NotNil(t, provider)

		ctx := context.Background()
		inputData := map[string]any{"key": "value"}

		newCtx, err := provider.AddDataToContext(ctx, inputData)

		assert.Error(t, err, "Should return error for static provider")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
		assert.True(t, errors.Is(err, ErrStaticProviderNoRuntimeUpdates))

		// Verify static data is still available
		data, getErr := provider.GetData(ctx)
		assert.NoError(t, getErr)
		assert.Equal(t, simpleData, data, "Static data should still be available")
	})

	t.Run("mixed providers (static fails, context succeeds)", func(t *testing.T) {
		provider := NewCompositeProvider(
			NewStaticProvider(simpleData),
			NewContextProvider(constants.EvalData),
		)
		require.NotNil(t, provider)

		ctx := context.Background()
		inputData := map[string]any{"key": "value"}

		newCtx, err := provider.AddDataToContext(ctx, inputData)

		assert.NoError(t, err, "Should not return error when at least one provider succeeds")
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		// Verify both static and context data are available
		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)

		// Static data should be present
		assert.Equal(t, simpleData["string"], data["string"], "Static data should be present")

		// Context data should be present
		assert.Contains(t, data, constants.InputData)
		inputDataResult, ok := data[constants.InputData].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "value", inputDataResult["key"], "Context data should be present")
	})

	t.Run("all providers fail", func(t *testing.T) {
		provider := NewCompositeProvider(
			NewStaticProvider(simpleData),
			newMockErrorProvider(),
		)
		require.NotNil(t, provider)

		ctx := context.Background()
		inputData := map[string]any{"key": "value"}

		newCtx, err := provider.AddDataToContext(ctx, inputData)

		assert.Error(t, err, "Should return error when all non-static providers fail")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
	})

	t.Run("nil providers are skipped", func(t *testing.T) {
		provider := NewCompositeProvider(
			nil,
			NewContextProvider(constants.EvalData),
			nil,
		)
		require.NotNil(t, provider)

		ctx := context.Background()
		inputData := map[string]any{"key": "value"}

		newCtx, err := provider.AddDataToContext(ctx, inputData)

		assert.NoError(t, err, "Should not return error when skipping nil providers")
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		// Verify context data was added
		data, err := provider.GetData(newCtx)
		assert.NoError(t, err)
		assert.Contains(t, data, constants.InputData)
	})

	t.Run("composite with only static providers", func(t *testing.T) {
		provider := NewCompositeProvider(
			NewStaticProvider(map[string]any{"key1": "value1"}),
			NewStaticProvider(map[string]any{"key2": "value2"}),
		)
		require.NotNil(t, provider)

		ctx := context.Background()
		inputData := map[string]any{"key": "value"}

		newCtx, err := provider.AddDataToContext(ctx, inputData)

		assert.Error(t, err, "Should return error when all providers are static")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
		assert.True(t, errors.Is(err, ErrStaticProviderNoRuntimeUpdates),
			"Error should be StaticProviderNoRuntimeUpdates")
	})
}

// TestCompositeProvider_NestedStructures tests deep nesting of composite providers
func TestCompositeProvider_NestedStructures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupProviders func() *CompositeProvider
		setupContext   func() context.Context
		expectedResult map[string]any
	}{
		{
			name: "nested composite providers with static data",
			setupProviders: func() *CompositeProvider {
				// Create the inner composite provider with two static providers
				innerStatic1 := NewStaticProvider(map[string]any{
					"inner1_key": "inner1_value",
					"shared_key": "inner1_value", // Will be overridden by inner2
				})
				innerStatic2 := NewStaticProvider(map[string]any{
					"inner2_key": "inner2_value",
					"shared_key": "inner2_value", // Will override inner1 but be overridden by outer
				})
				innerComposite := NewCompositeProvider(innerStatic1, innerStatic2)

				// Create the outer composite provider with one static provider and the inner composite
				outerStatic := NewStaticProvider(map[string]any{
					"outer_key":  "outer_value",
					"shared_key": "outer_value", // Will override both inner providers
				})
				return NewCompositeProvider(innerComposite, outerStatic)
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedResult: map[string]any{
				"inner1_key": "inner1_value",
				"inner2_key": "inner2_value",
				"outer_key":  "outer_value",
				"shared_key": "outer_value", // Verifies proper override: outer > inner2 > inner1
			},
		},
		{
			name: "deeply nested composite providers",
			setupProviders: func() *CompositeProvider {
				// Level 3 (deepest)
				level3Static := NewStaticProvider(map[string]any{
					"level":        3,
					"level3_key":   "level3_value",
					"override_key": "level3_value",
				})

				// Level 2
				level2Static := NewStaticProvider(map[string]any{
					"level":        2,
					"level2_key":   "level2_value",
					"override_key": "level2_value",
				})
				level2Composite := NewCompositeProvider(level3Static, level2Static)

				// Level 1 (outermost)
				level1Static := NewStaticProvider(map[string]any{
					"level":        1,
					"level1_key":   "level1_value",
					"override_key": "level1_value",
				})
				return NewCompositeProvider(level2Composite, level1Static)
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedResult: map[string]any{
				"level":        1, // Should be overridden to 1
				"level1_key":   "level1_value",
				"level2_key":   "level2_value",
				"level3_key":   "level3_value",
				"override_key": "level1_value", // Verifies proper override hierarchy
			},
		},
		{
			name: "mixed provider types in nested composites",
			setupProviders: func() *CompositeProvider {
				// Base static provider
				baseStatic := NewStaticProvider(map[string]any{
					"static_key": "static_value",
					"shared_key": "static_value",
				})

				// Context provider (will be populated in test)
				contextProvider := NewContextProvider(constants.EvalData)

				// Inner composite with static and context
				innerComposite := NewCompositeProvider(baseStatic, contextProvider)

				// Outer static to override some values
				outerStatic := NewStaticProvider(map[string]any{
					"outer_key":  "outer_value",
					"shared_key": "outer_value", // Will override both inner providers
				})

				return NewCompositeProvider(innerComposite, outerStatic)
			},
			setupContext: func() context.Context {
				data := map[string]any{
					"context_key": "context_value",
					constants.InputData: map[string]any{
						"nested_key": "nested_value",
					},
				}
				return context.WithValue(context.Background(), constants.EvalData, data)
			},
			expectedResult: map[string]any{
				"static_key":  "static_value",
				"outer_key":   "outer_value",
				"shared_key":  "outer_value",   // Outer static wins
				"context_key": "context_value", // From context
				constants.InputData: map[string]any{ // Nested under input_data from context
					"nested_key": "nested_value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			composite := tt.setupProviders()
			ctx := tt.setupContext()

			// Get the combined data
			result, err := composite.GetData(ctx)
			require.NoError(t, err, "GetData should not error with valid providers")

			// Verify all expected values are present
			assertMapContainsExpectedHelper(t, tt.expectedResult, result)
		})
	}
}

// TestCompositeProvider_DeepMerge tests the deep merge functionality
func TestCompositeProvider_DeepMerge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		src         map[string]any
		dst         map[string]any
		expected    map[string]any
		description string
	}{
		{
			name:        "empty maps",
			src:         map[string]any{},
			dst:         map[string]any{},
			expected:    map[string]any{},
			description: "Merging empty maps should result in an empty map",
		},
		{
			name:        "src empty, dst with data",
			src:         map[string]any{},
			dst:         map[string]any{"key": "value"},
			expected:    map[string]any{"key": "value"},
			description: "An empty source with a populated destination should use destination values",
		},
		{
			name:        "src with data, dst empty",
			src:         map[string]any{"key": "value"},
			dst:         map[string]any{},
			expected:    map[string]any{"key": "value"},
			description: "A populated source with an empty destination should keep source values",
		},
		{
			name: "non-map dst replaced by map",
			src: map[string]any{
				"key": "string_value",
			},
			dst: map[string]any{
				"key": map[string]any{"nested": "value"},
			},
			expected: map[string]any{
				"key": map[string]any{"nested": "value"},
			},
			description: "A non-map value should be completely replaced by a map value",
		},
		{
			name: "map src replaced by non-map",
			src: map[string]any{
				"key": map[string]any{"nested": "value"},
			},
			dst: map[string]any{
				"key": "string_value",
			},
			expected: map[string]any{
				"key": "string_value",
			},
			description: "A map value should be completely replaced by a non-map value",
		},
		{
			name: "arrays are replaced, not merged",
			src: map[string]any{
				"array": []any{1, 2, 3},
			},
			dst: map[string]any{
				"array": []any{4, 5},
			},
			expected: map[string]any{
				"array": []any{4, 5},
			},
			description: "Arrays should be completely replaced, not merged",
		},
		{
			name: "very deep nesting is handled correctly",
			src: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": map[string]any{
							"level4": map[string]any{
								"source": "value",
								"shared": "source",
							},
						},
					},
				},
			},
			dst: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": map[string]any{
							"level4": map[string]any{
								"dest":   "value",
								"shared": "dest",
							},
						},
					},
				},
			},
			expected: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": map[string]any{
							"level4": map[string]any{
								"source": "value", // Preserved from source
								"dest":   "value", // Added from destination
								"shared": "dest",  // Overridden by destination
							},
						},
					},
				},
			},
			description: "Deep nesting should merge correctly at all levels",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deepMerge(tt.src, tt.dst)
			assert.Equal(t, tt.expected, result, tt.description)

			// Verify source was not modified (should be a new map)
			srcCopy := make(map[string]any)
			for k, v := range tt.src {
				srcCopy[k] = v
			}
			assert.Equal(t, srcCopy, tt.src, "Source map should not be modified")
		})
	}
}
