package data

import (
	"context"
	"testing"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompositeProviderWithStaticAndContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		staticData     map[string]any
		contextData    map[string]any
		expectedResult map[string]any
	}{
		{
			name: "static data only",
			staticData: map[string]any{
				"static_key": "static_value",
			},
			contextData: nil,
			expectedResult: map[string]any{
				"static_key": "static_value",
			},
		},
		{
			name:       "context data only",
			staticData: nil,
			contextData: map[string]any{
				"runtime_key": "runtime_value",
			},
			expectedResult: map[string]any{
				"runtime_key": "runtime_value",
			},
		},
		{
			name: "both static and context data with no overlap",
			staticData: map[string]any{
				"static_key": "static_value",
			},
			contextData: map[string]any{
				"runtime_key": "runtime_value",
			},
			expectedResult: map[string]any{
				"static_key":  "static_value",
				"runtime_key": "runtime_value",
			},
		},
		{
			name: "both static and context data with overlap (context wins)",
			staticData: map[string]any{
				"shared_key": "static_value",
				"static_key": "static_value",
			},
			contextData: map[string]any{
				"shared_key":  "runtime_value",
				"runtime_key": "runtime_value",
			},
			expectedResult: map[string]any{
				"shared_key":  "runtime_value", // Context provider overrides static provider
				"static_key":  "static_value",
				"runtime_key": "runtime_value",
			},
		},
		{
			name: "nested data structures",
			staticData: map[string]any{
				"config": map[string]any{
					"timeout": 30,
					"retries": 3,
				},
			},
			contextData: map[string]any{
				"input": "API User",
				"request": map[string]any{
					"id": "123",
				},
			},
			expectedResult: map[string]any{
				"config": map[string]any{
					"timeout": 30,
					"retries": 3,
				},
				"input": "API User",
				"request": map[string]any{
					"id": "123",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create the static provider
			staticProvider := NewStaticProvider(tt.staticData)

			// Create a context provider with a test key
			contextKey := constants.ContextKey("test_key")
			contextProvider := NewContextProvider(contextKey)

			// Create a composite provider with the static provider first and context provider second
			// This order means the context provider will override the static provider for duplicate keys
			composite := NewCompositeProvider(staticProvider, contextProvider)

			// Create a context with the test data
			ctx := context.Background()
			if tt.contextData != nil {
				ctx = context.WithValue(ctx, contextKey, tt.contextData)
			}

			// Get the combined data
			result, err := composite.GetData(ctx)
			require.NoError(t, err)

			// Verify the result contains the expected data
			if tt.expectedResult != nil {
				for key, expectedValue := range tt.expectedResult {
					assert.Contains(t, result, key)
					assert.Equal(t, expectedValue, result[key])
				}
			}
		})
	}
}

// TestCompositePrepareContext tests that the AddDataToContext method correctly
// distributes data to all providers in the chain
func TestCompositePrepareContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		setupProviders   func() []Provider
		inputData        []any
		expectAllSuccess bool
		expectPartial    bool
	}{
		{
			name: "all providers succeed",
			setupProviders: func() []Provider {
				return []Provider{
					NewContextProvider(constants.ContextKey("key1")),
					NewContextProvider(constants.ContextKey("key2")),
				}
			},
			inputData: []any{
				map[string]any{"data": "value1"},
				map[string]any{"data": "value2"},
			},
			expectAllSuccess: true,
			expectPartial:    false,
		},
		{
			name: "static provider fails but context provider succeeds",
			setupProviders: func() []Provider {
				return []Provider{
					NewStaticProvider(
						map[string]any{"static": "data"},
					), // Will fail on AddDataToContext
					NewContextProvider(constants.ContextKey("runtime")),
				}
			},
			inputData: []any{
				map[string]any{"data": "value"},
			},
			expectAllSuccess: true, // Updated: StaticProvider errors are now ignored when ContextProvider succeeds
			expectPartial:    true, // The context provider should still succeed
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			providers := tt.setupProviders()
			composite := NewCompositeProvider(providers...)

			ctx := context.Background()
			newCtx, err := composite.AddDataToContext(ctx, tt.inputData...)

			if tt.expectAllSuccess {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			// Even with errors, the context should be updated if any provider succeeded
			if tt.expectPartial {
				assert.NotEqual(t, ctx, newCtx)
			}
		})
	}
}

func TestNestedCompositeProviders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupProviders func() *CompositeProvider
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
			expectedResult: map[string]any{
				"level":        1, // Should be overridden to 1
				"level1_key":   "level1_value",
				"level2_key":   "level2_value",
				"level3_key":   "level3_value",
				"override_key": "level1_value", // Verifies proper override hierarchy
			},
		},
		{
			name: "composite with nil providers should be skipped",
			setupProviders: func() *CompositeProvider {
				// Create providers including nil, which should be skipped
				static1 := NewStaticProvider(map[string]any{
					"key1": "value1",
				})
				// Add a nil provider in the mix
				return NewCompositeProvider(static1, nil, NewStaticProvider(map[string]any{
					"key2": "value2",
				}))
			},
			expectedResult: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "nested composites with complex nested data structures",
			setupProviders: func() *CompositeProvider {
				// Inner provider with nested map
				innerProvider := NewStaticProvider(map[string]any{
					"config": map[string]any{
						"database": map[string]any{
							"host":     "localhost",
							"port":     5432,
							"username": "user1",
							"timeout":  30,
						},
						"cache": map[string]any{
							"enabled": true,
							"ttl":     60,
						},
					},
					"metrics": map[string]any{
						"enabled": false,
					},
				})

				// Outer provider that overrides some nested values
				outerProvider := NewStaticProvider(map[string]any{
					"config": map[string]any{
						"database": map[string]any{
							"username": "admin",  // Should override the inner value
							"password": "secret", // New field
						},
						"logging": map[string]any{ // New nested section
							"level": "debug",
						},
					},
					"metrics": map[string]any{
						"enabled":  true, // Override the inner value
						"interval": 15,   // New field
					},
				})

				return NewCompositeProvider(innerProvider, outerProvider)
			},
			expectedResult: map[string]any{
				"config": map[string]any{
					"database": map[string]any{
						"username": "admin",     // Overridden
						"password": "secret",    // Added
						"host":     "localhost", // Preserved
						"port":     5432,        // Preserved
						"timeout":  30,          // Preserved
					},
					"cache": map[string]any{
						"enabled": true,
						"ttl":     60,
					},
					"logging": map[string]any{
						"level": "debug",
					},
				},
				"metrics": map[string]any{
					"enabled":  true, // Overridden
					"interval": 15,   // Added
				},
			},
		},
		{
			name: "multiple nested composite providers with mixed order",
			setupProviders: func() *CompositeProvider {
				// Create base providers
				baseA := NewStaticProvider(map[string]any{"keyA": "valueA", "shared": "baseA"})
				baseB := NewStaticProvider(map[string]any{"keyB": "valueB", "shared": "baseB"})
				baseC := NewStaticProvider(map[string]any{"keyC": "valueC", "shared": "baseC"})

				// Create first level composites
				compositeAB := NewCompositeProvider(baseA, baseB) // B overrides A

				// Create second level with mixed order
				// Order: (A->B)->C
				return NewCompositeProvider(compositeAB, baseC) // C overrides AB
			},
			expectedResult: map[string]any{
				"keyA":   "valueA",
				"keyB":   "valueB",
				"keyC":   "valueC",
				"shared": "baseC", // C should override both A and B
			},
		},
		{
			name: "composite provider with multiple levels of array data",
			setupProviders: func() *CompositeProvider {
				// First provider with array data
				provider1 := NewStaticProvider(map[string]any{
					"array": []any{1, 2, 3},
					"nestedArrays": map[string]any{
						"numbers": []any{1, 2, 3},
					},
				})

				// Second provider with different array data (will replace the arrays)
				provider2 := NewStaticProvider(map[string]any{
					"array": []any{4, 5, 6},
					"nestedArrays": map[string]any{
						"letters": []any{"a", "b", "c"},
					},
				})

				// Third provider with more array modifications
				provider3 := NewStaticProvider(map[string]any{
					"nestedArrays": map[string]any{
						"numbers": []any{
							7,
							8,
							9,
						}, // Should replace the numbers array from provider1
					},
				})

				// Composite providers at different levels
				innerComposite := NewCompositeProvider(provider1, provider2)
				return NewCompositeProvider(innerComposite, provider3)
			},
			expectedResult: map[string]any{
				"array": []any{4, 5, 6}, // From provider2 (overrides provider1)
				"nestedArrays": map[string]any{
					"numbers": []any{7, 8, 9},       // From provider3 (overrides provider1)
					"letters": []any{"a", "b", "c"}, // From provider2 (preserved)
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			composite := tt.setupProviders()
			ctx := context.Background()

			// Get the combined data
			result, err := composite.GetData(ctx)
			require.NoError(t, err)

			// Verify the result matches expected values
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
