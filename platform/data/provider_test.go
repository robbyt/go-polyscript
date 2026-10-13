package data

import (
	"context"
	"sync"
	"testing"

	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		provider := NewStaticProvider(simpleData)
		ctx := t.Context()

		result, err := provider.GetData(ctx)
		require.NoError(t, err)
		assert.Equal(t, simpleData, result)

		// Get a fresh copy to verify data consistency
		newResult, err := provider.GetData(ctx)
		require.NoError(t, err)
		assert.Equal(t, simpleData, newResult)
	})

	t.Run("static provider with empty data", func(t *testing.T) {
		provider := NewStaticProvider(nil)
		ctx := t.Context()

		result, err := provider.GetData(ctx)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	// Test context provider
	t.Run("context provider with valid data", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.WithValue(t.Context(), constants.EvalData, simpleData)

		result, err := provider.GetData(ctx)
		require.NoError(t, err)
		assert.Equal(t, simpleData, result)

		// Get a fresh copy to verify data consistency
		newResult, err := provider.GetData(ctx)
		require.NoError(t, err)
		assert.Equal(t, simpleData, newResult)
	})

	t.Run("context provider with empty key", func(t *testing.T) {
		provider := NewContextProvider("")
		ctx := t.Context()

		result, err := provider.GetData(ctx)
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("context provider with invalid value type", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := context.WithValue(t.Context(), constants.EvalData, "not a map")

		result, err := provider.GetData(ctx)
		require.Error(t, err)
		assert.Nil(t, result)
	})

	// Test composite provider
	t.Run("composite provider with multiple sources", func(t *testing.T) {
		provider := NewCompositeProvider(
			NewStaticProvider(map[string]any{"static": "value", "shared": "static"}),
			NewContextProvider(constants.EvalData),
		)

		ctx := context.WithValue(
			t.Context(),
			constants.EvalData,
			map[string]any{"context": "value", "shared": "context"},
		)

		result, err := provider.GetData(ctx)
		require.NoError(t, err)

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
		require.NoError(t, err)
		assert.Equal(t, result, newResult)
	})

	t.Run("empty composite provider", func(t *testing.T) {
		provider := NewCompositeProvider()
		ctx := t.Context()

		result, err := provider.GetData(ctx)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("composite provider with error", func(t *testing.T) {
		provider := NewCompositeProvider(
			NewStaticProvider(simpleData),
			newMockErrorProvider(),
		)
		ctx := t.Context()

		result, err := provider.GetData(ctx)
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestProvider_AddDataToContext tests adding data to context across provider implementations
func TestProvider_AddDataToContext(t *testing.T) {
	t.Parallel()

	// Test with static provider
	t.Run("static provider should reject all data", func(t *testing.T) {
		provider := NewStaticProvider(simpleData)
		ctx := t.Context()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key": "value"})

		require.Error(t, err, "StaticProvider should reject data additions")
		require.ErrorIs(t, err, ErrStaticProviderNoRuntimeUpdates)
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")

		// Verify static data is still available
		data, err := provider.GetData(ctx)
		require.NoError(t, err)
		assert.Equal(t, simpleData, data)
	})

	// Test with context provider
	t.Run("context provider with valid map data", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := t.Context()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key": "value"})

		require.NoError(t, err)
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		// Verify data was stored correctly
		data, err := provider.GetData(newCtx)
		require.NoError(t, err)

		// Data should be at root level with no namespace
		assert.Equal(t, "value", data["key"], "Data should be at root level")
	})

	t.Run("context provider with HTTP request", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := t.Context()
		req := createTestRequestHelper()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"request": req})

		require.NoError(t, err)
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		// Verify request data was stored correctly
		data, err := provider.GetData(newCtx)
		require.NoError(t, err)

		// Request should be at root level with direct access
		requestMap, ok := data["request"].(map[string]any)
		assert.True(t, ok, "Request data should be a map at root level")
		assert.Equal(t, "GET", requestMap["Method"])
		assert.Equal(t, "/test", requestMap["URL_Path"])
	})

	t.Run("context provider with empty key", func(t *testing.T) {
		provider := NewContextProvider("")
		ctx := t.Context()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key": "value"})

		require.Error(t, err, "Empty context key should cause error")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged on error")
	})

	// Test with composite provider
	t.Run("composite provider with mixed providers", func(t *testing.T) {
		provider := NewCompositeProvider(
			NewStaticProvider(simpleData),
			NewContextProvider(constants.EvalData),
		)
		ctx := t.Context()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key": "value"})

		require.NoError(
			t,
			err,
			"StaticProvider error should be ignored when ContextProvider succeeds",
		)
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		// Verify data
		data, err := provider.GetData(newCtx)
		require.NoError(t, err)

		// Should have both static and context data
		assert.Equal(t, simpleData["string"], data["string"], "Should contain static data")
		assert.Equal(t, "value", data["key"], "Should contain added data")
	})

	t.Run("composite provider with all failures", func(t *testing.T) {
		provider := NewCompositeProvider(
			NewStaticProvider(simpleData),
			newMockErrorProvider(),
		)
		ctx := t.Context()

		newCtx, err := provider.AddDataToContext(ctx, map[string]any{"key": "value"})

		require.Error(t, err, "Should error when all providers fail")
		assert.Equal(t, ctx, newCtx, "Context should remain unchanged")
	})

	// Test that multiple keys in a single map work
	t.Run("context provider with multiple keys in one map", func(t *testing.T) {
		provider := NewContextProvider(constants.EvalData)
		ctx := t.Context()

		newCtx, err := provider.AddDataToContext(ctx,
			map[string]any{"key1": "value1", "key2": "value2"})

		require.NoError(t, err)
		assert.NotEqual(t, ctx, newCtx, "Context should be modified")

		// Verify data was merged correctly
		data, err := provider.GetData(newCtx)
		require.NoError(t, err)

		assert.Equal(t, "value1", data["key1"], "Should contain first item")
		assert.Equal(t, "value2", data["key2"], "Should contain second item")
	})
}

// TestProvider_DeepMerge tests the deep merge functionality specifically
func TestProvider_DeepMerge(t *testing.T) {
	t.Parallel()

	t.Run("simple merge with no overlaps", func(t *testing.T) {
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
		src := map[string]any{"array": []string{"one", "two", "three"}}
		dst := map[string]any{"array": []string{"four", "five"}}
		expected := map[string]any{"array": []string{"four", "five"}}

		result := deepMerge(src, dst)
		assert.Equal(t, expected, result)
	})

	t.Run("empty maps", func(t *testing.T) {
		result1 := deepMerge(map[string]any{}, map[string]any{"key": "value"})
		assert.Equal(t, map[string]any{"key": "value"}, result1)

		result2 := deepMerge(map[string]any{"key": "value"}, map[string]any{})
		assert.Equal(t, map[string]any{"key": "value"}, result2)
	})

	t.Run("original maps should not be modified", func(t *testing.T) {
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

// TestProvider_GetData_ConcurrentMutation is the -race regression test for
// #159. Reader goroutines mutate their GetData results in place while writer
// goroutines derive new contexts from the same parent with AddDataToContext.
// When GetData handed out shared storage this raced and corrupted the
// provider's data; with deep copies every goroutine owns its own maps.
func TestProvider_GetData_ConcurrentMutation(t *testing.T) {
	t.Parallel()

	static := NewStaticProvider(map[string]any{"static": nestedFixtureHelper()})
	ctxProvider := NewContextProvider(constants.EvalData)
	composite := NewCompositeProvider(static, ctxProvider)

	parent, err := composite.AddDataToContext(t.Context(), nestedFixtureHelper())
	require.NoError(t, err)

	wantComposite := nestedFixtureHelper()
	wantComposite["static"] = nestedFixtureHelper()
	readers := []concurrentReader{
		{"static", static, map[string]any{"static": nestedFixtureHelper()}},
		{"context", ctxProvider, nestedFixtureHelper()},
		{"composite", composite, wantComposite},
	}

	const goroutinesPerRole = 8
	const iterations = 50

	var wg sync.WaitGroup
	for _, r := range readers {
		for range goroutinesPerRole {
			wg.Go(func() { mutateGetDataHelper(t, parent, r, iterations) })
		}
	}
	for range goroutinesPerRole {
		wg.Go(func() { addAndVerifyHelper(t, parent, composite, iterations) })
	}
	wg.Wait()

	for _, r := range readers {
		got, err := r.getter.GetData(parent)
		require.NoError(t, err)
		assert.Equal(t, r.want, got, "%s provider state must be unchanged", r.name)
	}
}

// concurrentReader is a provider that TestProvider_GetData_ConcurrentMutation
// reads from concurrently, with the data it must still return afterwards.
type concurrentReader struct {
	name   string
	getter Getter
	want   map[string]any
}

// mutateGetDataHelper calls GetData on parent iterations times and applies
// every fixtureMutationsHelper mutation to each fixture-shaped map in the result.
// It runs on a non-test goroutine, so failures use t.Errorf, never FailNow.
func mutateGetDataHelper(t *testing.T, parent context.Context, r concurrentReader, iterations int) {
	t.Helper()
	for range iterations {
		got, err := r.getter.GetData(parent)
		if err != nil {
			t.Errorf("%s GetData: %v", r.name, err)
			return
		}
		for _, target := range fixtureTargetsHelper(got) {
			for _, m := range fixtureMutationsHelper {
				m.mutate(target)
			}
		}
	}
}

// fixtureTargetsHelper returns got and its "static" sub-map, keeping only the
// maps shaped like nestedFixtureHelper (those with a "cfg" key).
func fixtureTargetsHelper(got map[string]any) []map[string]any {
	candidates := []map[string]any{got}
	if nested, ok := got["static"].(map[string]any); ok {
		candidates = append(candidates, nested)
	}
	var targets []map[string]any
	for _, c := range candidates {
		if _, ok := c["cfg"]; ok {
			targets = append(targets, c)
		}
	}
	return targets
}

// addAndVerifyHelper derives a new context from parent iterations times with
// AddDataToContext and checks that GetData on it returns the value just added.
// It runs on a non-test goroutine, so failures use t.Errorf, never FailNow.
func addAndVerifyHelper(t *testing.T, parent context.Context, composite *CompositeProvider, iterations int) {
	t.Helper()
	for j := range iterations {
		next, err := composite.AddDataToContext(parent, map[string]any{
			"cfg": map[string]any{"x": j},
		})
		if err != nil {
			t.Errorf("AddDataToContext iter %d: %v", j, err)
			return
		}
		got, err := composite.GetData(next)
		if err != nil {
			t.Errorf("GetData iter %d: %v", j, err)
			return
		}
		cfg, ok := got["cfg"].(map[string]any)
		if !assert.True(t, ok, "cfg should be a map") {
			return
		}
		assert.Equal(t, j, cfg["x"])
	}
}
