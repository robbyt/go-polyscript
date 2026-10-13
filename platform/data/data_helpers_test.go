package data

import (
	"context"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Standard test data sets used across all provider tests
var (
	// Simple data for testing basic functionality
	simpleData = map[string]any{
		"string": "value",
		"int":    42,
		"bool":   true,
	}

	// Complex data for testing nested structures
	complexData = map[string]any{
		"string": "value",
		"int":    42,
		"bool":   true,
		"nested": map[string]any{
			"key":   "nested value",
			"inner": map[string]any{"deep": "very deep"},
		},
		"array": []string{"one", "two", "three"},
	}
)

// createTestRequestHelper creates a standard HTTP request for testing
func createTestRequestHelper() *http.Request {
	return &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/test", RawQuery: "param=value"},
		Header: http.Header{"Content-Type": []string{"application/json"}},
	}
}

// MockProvider is a testify mock implementation of Provider
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) GetData(ctx context.Context) (map[string]any, error) {
	args := m.Called(ctx)
	data, _ := args.Get(0).(map[string]any)
	return data, args.Error(1)
}

func (m *MockProvider) AddDataToContext(
	ctx context.Context,
	data map[string]any,
) (context.Context, error) {
	args := m.Called(ctx, data)
	newCtx, _ := args.Get(0).(context.Context)
	return newCtx, args.Error(1)
}

// baseCtxKey keys the marker value stored by newBaseCtxHelper.
type baseCtxKey struct{}

// newBaseCtxHelper returns a pointer-backed context carrying a marker
// value, so error-path tests can check with assert.Same that a provider
// returned exactly the ctx it was given rather than one derived from it.
func newBaseCtxHelper(t *testing.T) context.Context {
	t.Helper()
	return context.WithValue(t.Context(), baseCtxKey{}, "base")
}

// countingProvider wraps a Provider and counts its AddDataToContext calls,
// so tests can check that a CompositeProvider still called every provider.
type countingProvider struct {
	Provider
	calls atomic.Int32
}

func (p *countingProvider) AddDataToContext(
	ctx context.Context,
	data map[string]any,
) (context.Context, error) {
	p.calls.Add(1)
	return p.Provider.AddDataToContext(ctx, data)
}

// newMockErrorProvider creates a mock provider that returns errors
func newMockErrorProvider() *MockProvider {
	provider := new(MockProvider)
	provider.On("GetData", mock.Anything).Return(nil, assert.AnError)
	provider.On("AddDataToContext", mock.Anything, mock.Anything).
		Return(mock.Anything, assert.AnError)
	return provider
}

// assertMapContainsExpectedHelper recursively asserts that a map contains all expected key/value pairs
func assertMapContainsExpectedHelper(t *testing.T, expected, actual map[string]any) {
	t.Helper()
	for key, expectedValue := range expected {
		assert.Contains(t, actual, key, "Result should contain key: %s", key)

		// Handle nested maps recursively
		expectedMap, expectedIsMap := expectedValue.(map[string]any)
		actualValue, exists := actual[key]
		require.True(t, exists, "Key should exist: %s", key)

		actualMap, actualIsMap := actualValue.(map[string]any)

		if expectedIsMap && actualIsMap {
			assertMapContainsExpectedHelper(t, expectedMap, actualMap)
		} else {
			assert.Equal(t, expectedValue, actualValue, "Value should match for key: %s", key)
		}
	}
}

// getDataCheckHelper checks if multiple calls to GetData return consistent results
func getDataCheckHelper(t *testing.T, provider Provider, ctx context.Context) {
	t.Helper()
	result1, err1 := provider.GetData(ctx)
	require.NoError(t, err1)

	result2, err2 := provider.GetData(ctx)
	require.NoError(t, err2)

	assert.Equal(t, result1, result2, "Multiple GetData calls should return consistent results")
}

// nestedFixtureHelper returns a fresh map with nested maps and slices, for
// tests that check a provider's GetData result shares no mutable storage
// with the provider.
func nestedFixtureHelper() map[string]any {
	return map[string]any{
		"top": "value",
		"cfg": map[string]any{
			"x":     1,
			"inner": map[string]any{"deep": "value"},
		},
		"list": []any{"a", map[string]any{"k": "v"}},
	}
}

// fixtureMutationsHelper are in-place mutations of a map shaped like
// nestedFixtureHelper, at every nesting level.
var fixtureMutationsHelper = []struct {
	name   string
	mutate func(m map[string]any)
}{
	{"top-level write", func(m map[string]any) { m["top"] = "hacked" }},
	{"top-level add", func(m map[string]any) { m["added"] = "hacked" }},
	{"top-level delete", func(m map[string]any) { delete(m, "top") }},
	{"nested map write", func(m map[string]any) {
		m["cfg"].(map[string]any)["x"] = 999
	}},
	{"deeply nested map write", func(m map[string]any) {
		m["cfg"].(map[string]any)["inner"].(map[string]any)["deep"] = "hacked"
	}},
	{"slice element write", func(m map[string]any) {
		m["list"].([]any)[0] = "hacked"
	}},
	{"map inside slice write", func(m map[string]any) {
		m["list"].([]any)[1].(map[string]any)["k"] = "hacked"
	}},
}

// assertGetDataIsolatedHelper mutates the result of getter.GetData(ctx) in
// every way listed in fixtureMutationsHelper and asserts that a later
// GetData still returns want.
func assertGetDataIsolatedHelper(
	t *testing.T,
	getter Getter,
	ctx context.Context,
	want map[string]any,
) {
	t.Helper()
	for _, m := range fixtureMutationsHelper {
		t.Run(m.name, func(t *testing.T) {
			got, err := getter.GetData(ctx)
			require.NoError(t, err)
			require.Equal(t, want, got, "GetData should start from the expected data")

			m.mutate(got)

			after, err := getter.GetData(ctx)
			require.NoError(t, err)
			assert.Equal(t, want, after, "mutating a GetData result must not change provider state")
		})
	}
}
