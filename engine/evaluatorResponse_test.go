package engine_test

import (
	"testing"

	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEvaluatorResponseInterface tests all methods of the EvaluatorResponse interface
func TestEvaluatorResponseInterface(t *testing.T) {
	t.Parallel()
	mockResponse := new(mocks.EvaluatorResponse)

	// Test Type method with various return types
	t.Run("Type method", func(t *testing.T) {
		typeTests := []struct {
			name     string
			dataType data.Types
		}{
			{"String type", data.STRING},
			{"Integer type", data.INT},
			{"Float type", data.FLOAT},
			{"Boolean type", data.BOOL},
			{"Map type", data.MAP},
			{"Function type", data.FUNCTION},
			{"List type", data.LIST},
			{"Set type", data.SET},
			{"Tuple type", data.TUPLE},
			{"Error type", data.ERROR},
			{"None type", data.NONE},
		}

		for _, tt := range typeTests {
			t.Run(tt.name, func(t *testing.T) {
				mockResponse.On("Type").Return(tt.dataType).Once()
				result := mockResponse.Type()
				assert.Equal(t, tt.dataType, result, "Type() should return expected type")
			})
		}
	})

	t.Run("Inspect method", func(t *testing.T) {
		inspectTests := []struct {
			name          string
			inspectResult string
		}{
			{"Empty string", ""},
			{"Simple string", "test string"},
			{"JSON representation", `{"key":"value"}`},
			{"Integer representation", "42"},
			{"Boolean representation", "true"},
		}

		for _, tt := range inspectTests {
			t.Run(tt.name, func(t *testing.T) {
				mockResponse.On("Inspect").Return(tt.inspectResult).Once()
				result := mockResponse.Inspect()
				assert.Equal(
					t,
					tt.inspectResult,
					result,
					"Inspect() should return expected string representation",
				)
			})
		}
	})

	// Test Interface method with different return types
	t.Run("Interface method", func(t *testing.T) {
		interfaceTests := []struct {
			name  string
			value any
		}{
			{"String value", "test string"},
			{"Integer value", 42},
			{"Float value", 3.14},
			{"Boolean value", true},
			{"Map value", map[string]any{"key": "value"}},
			{"Slice value", []any{1, 2, 3}},
			{"Nil value", nil},
		}

		for _, tt := range interfaceTests {
			t.Run(tt.name, func(t *testing.T) {
				mockResponse.On("Interface").Return(tt.value).Once()
				result := mockResponse.Interface()
				assert.Equal(t, tt.value, result, "Interface() should return expected value")
			})
		}
	})

	// Test script ID and execution time methods
	t.Run("Script metadata methods", func(t *testing.T) {
		mockResponse.On("GetScriptExeID").Return("script-123").Once()
		scriptID := mockResponse.GetScriptExeID()
		assert.Equal(t, "script-123", scriptID, "GetScriptExeID() should return expected ID")

		mockResponse.On("GetExecTime").Return("42ms").Once()
		execTime := mockResponse.GetExecTime()
		assert.Equal(t, "42ms", execTime, "GetExecTime() should return expected time")
	})

	// Verify all expected assertions
	mockResponse.AssertExpectations(t)
}

// TestEvaluatorResponseUsage tests how EvaluatorResponse is typically used in real code
func TestEvaluatorResponseUsage(t *testing.T) {
	t.Parallel()
	mockResponse := new(mocks.EvaluatorResponse)

	// Test a typical usage pattern where a string value is returned
	mockResponse.On("Interface").Return("Hello World").Once()
	mockResponse.On("Type").Return(data.STRING).Once()

	// Type checking pattern
	result := mockResponse.Interface()
	require.Equal(t, mockResponse.Type(), data.STRING)

	strResult, ok := result.(string)
	assert.True(t, ok, "Should convert to string")
	assert.Equal(t, "Hello World", strResult, "String value should match")

	// Test map pattern
	mapValue := map[string]any{
		"name": "John",
		"age":  42,
	}
	mockResponse.On("Interface").Return(mapValue).Once()
	mockResponse.On("Type").Return(data.MAP).Once()

	// Type checking for map
	result = mockResponse.Interface()
	require.Equal(t, mockResponse.Type(), data.MAP)

	mapResult, ok := result.(map[string]any)
	assert.True(t, ok, "Should convert to map")
	assert.Equal(t, mapValue, mapResult, "Map value should match")
	assert.Equal(t, "John", mapResult["name"], "Can access map values")

	// Verify all expected assertions
	mockResponse.AssertExpectations(t)
}
