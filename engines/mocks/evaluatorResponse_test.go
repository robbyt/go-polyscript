package mocks

import (
	"testing"

	"github.com/robbyt/go-polyscript/abstract/data"
	"github.com/robbyt/go-polyscript/abstract/evaluation"
	"github.com/stretchr/testify/assert"
)

// TestEvaluatorResponseImplementsInterface verifies at compile time
// that our mock EvaluatorResponse implements the evaluation.EvaluatorResponse interface.
func TestEvaluatorResponseImplementsInterface(t *testing.T) {
	t.Parallel()
	// This is a compile-time check - if it doesn't compile, the test fails
	var _ evaluation.EvaluatorResponse = (*EvaluatorResponse)(nil)
}

// TestEvaluatorResponseType tests the Type method for different value types
func TestEvaluatorResponseType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		mockVal  any
		expected data.Types
	}{
		{
			name:     "boolean value",
			mockVal:  true,
			expected: data.BOOL,
		},
		{
			name:     "integer value",
			mockVal:  42,
			expected: data.INT,
		},
		{
			name:     "string value",
			mockVal:  "test",
			expected: data.STRING,
		},
		{
			name:     "map value",
			mockVal:  map[string]any{"key": "value"},
			expected: data.MAP,
		},
		{
			name:     "direct data.Types value",
			mockVal:  data.FLOAT,
			expected: data.FLOAT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the mock
			mockResp := new(EvaluatorResponse)

			// Set up expectations
			mockResp.On("Type").Return(tt.mockVal)

			// Call the method
			result := mockResp.Type()

			// Verify the result
			assert.Equal(t, tt.expected, result)

			// Verify the mock expectations were met
			mockResp.AssertExpectations(t)
		})
	}
}

// TestEvaluatorResponseInspect tests the Inspect method
func TestEvaluatorResponseInspect(t *testing.T) {
	t.Parallel()
	mockResp := new(EvaluatorResponse)
	expected := "test string representation"

	// Set up expectations
	mockResp.On("Inspect").Return(expected)

	// Call the method
	result := mockResp.Inspect()

	// Verify the result
	assert.Equal(t, expected, result)

	// Verify the mock expectations were met
	mockResp.AssertExpectations(t)
}

// TestEvaluatorResponseInterface tests the Interface method
func TestEvaluatorResponseInterface(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		mockVal  any
		expected any
	}{
		{
			name:     "string value",
			mockVal:  "test",
			expected: "test",
		},
		{
			name:     "int value",
			mockVal:  42,
			expected: 42,
		},
		{
			name:     "map value",
			mockVal:  map[string]any{"key": "value"},
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "nil value",
			mockVal:  nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the mock
			mockResp := new(EvaluatorResponse)

			// Set up expectations
			mockResp.On("Interface").Return(tt.mockVal)

			// Call the method
			result := mockResp.Interface()

			// Verify the result
			assert.Equal(t, tt.expected, result)

			// Verify the mock expectations were met
			mockResp.AssertExpectations(t)
		})
	}
}

// TestEvaluatorResponseScriptExeID tests the GetScriptExeID method
func TestEvaluatorResponseScriptExeID(t *testing.T) {
	t.Parallel()
	mockResp := new(EvaluatorResponse)
	expected := "script-v1.0.0"

	// Set up expectations
	mockResp.On("GetScriptExeID").Return(expected)

	// Call the method
	result := mockResp.GetScriptExeID()

	// Verify the result
	assert.Equal(t, expected, result)

	// Verify the mock expectations were met
	mockResp.AssertExpectations(t)
}

// TestEvaluatorResponseExecTime tests the GetExecTime method
func TestEvaluatorResponseExecTime(t *testing.T) {
	t.Parallel()
	mockResp := new(EvaluatorResponse)
	expected := "100ms"

	// Set up expectations
	mockResp.On("GetExecTime").Return(expected)

	// Call the method
	result := mockResp.GetExecTime()

	// Verify the result
	assert.Equal(t, expected, result)

	// Verify the mock expectations were met
	mockResp.AssertExpectations(t)
}

// TestEvaluatorResponsePanicOnInvalidType tests the Type method when an invalid type is provided
func TestEvaluatorResponsePanicOnInvalidType(t *testing.T) {
	t.Parallel()
	// Create the mock
	mockResp := new(EvaluatorResponse)

	// Set up expectations with a complex value that doesn't have a direct type mapping
	mockResp.On("Type").Return(struct{ foo string }{"bar"})

	// The Type method should panic when given an unknown type
	assert.Panics(t, func() {
		mockResp.Type()
	})
}

// TestEvaluatorResponseFullUsage tests all methods together in a realistic usage scenario
func TestEvaluatorResponseFullUsage(t *testing.T) {
	t.Parallel()
	// Create the mock
	mockResp := new(EvaluatorResponse)

	// Set up expectations for all methods
	mockResp.On("Type").Return(data.STRING)
	mockResp.On("Inspect").Return("\"Hello, World!\"")
	mockResp.On("Interface").Return("Hello, World!")
	mockResp.On("GetScriptExeID").Return("script-v1.2.3")
	mockResp.On("GetExecTime").Return("50ms")

	// Verify the implementation satisfies the interface
	var response evaluation.EvaluatorResponse = mockResp

	// Call all methods
	assert.Equal(t, data.STRING, response.Type())
	assert.Equal(t, "\"Hello, World!\"", response.Inspect())
	assert.Equal(t, "Hello, World!", response.Interface())
	assert.Equal(t, "script-v1.2.3", response.GetScriptExeID())
	assert.Equal(t, "50ms", response.GetExecTime())

	// Verify all expectations were met
	mockResp.AssertExpectations(t)
}
