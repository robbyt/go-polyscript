package data

import (
	"context"
	"net/http"
	"net/url"
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

func (m *MockProvider) AddDataToContext(ctx context.Context, data ...any) (context.Context, error) {
	args := m.Called(append([]any{ctx}, data...))
	newCtx, _ := args.Get(0).(context.Context)
	return newCtx, args.Error(1)
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
