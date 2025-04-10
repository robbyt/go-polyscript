package evaluator

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewEvalResultNilHandler tests the behavior when a nil handler is provided
func TestNewEvalResultNilHandler(t *testing.T) {
	// Create with nil handler
	result := newEvalResult(nil, "test value", 100*time.Millisecond, "test-id")

	// Should create default handler and logger
	require.NotNil(t, result)
	require.NotNil(t, result.logHandler)
	require.NotNil(t, result.logger)

	// Should still store all values correctly
	assert.Equal(t, "test value", result.value)
	assert.Equal(t, 100*time.Millisecond, result.execTime)
	assert.Equal(t, "test-id", result.scriptExeID)
}

// TestExecResult_UnknownType tests handling of unknown value types
func TestExecResult_UnknownType(t *testing.T) {
	// Create a custom type
	type CustomType struct {
		Field string
	}

	// Create result with unknown type
	customValue := CustomType{Field: "test"}
	handler := slog.NewTextHandler(os.Stdout, nil)
	result := newEvalResult(handler, customValue, time.Second, "test-id")

	// Should return ERROR for unknown types
	assert.Equal(t, data.ERROR, result.Type())
}

// TestExecResult_Interface tests the Interface method
func TestExecResult_Interface(t *testing.T) {
	tests := []struct {
		name          string
		value         any
		expectedValue any
	}{
		{"nil value", nil, nil},
		{"string value", "test", "test"},
		{"int value", 42, 42},
		{"bool value", true, true},
		{"map value", map[string]any{"key": "value"}, map[string]any{"key": "value"}},
		{"list value", []any{1, 2, 3}, []any{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			result := newEvalResult(handler, tt.value, time.Second, "test-id")

			// Interface should return the original value
			assert.Equal(t, tt.expectedValue, result.Interface())
		})
	}
}

// TestExecResult_GetMetadata tests the metadata methods
func TestExecResult_GetMetadata(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	execTime := 123 * time.Millisecond
	scriptID := "test-script-9876"

	result := newEvalResult(handler, "test", execTime, scriptID)

	// Test GetScriptExeID
	assert.Equal(t, scriptID, result.GetScriptExeID())

	// Test GetExecTime
	assert.Equal(t, execTime.String(), result.GetExecTime())
}

// TestExecResult_InspectWithInvalidJSON tests what happens when JSON marshaling fails
func TestExecResult_InspectWithInvalidJSON(t *testing.T) {
	// Create a map with a value that can't be marshaled to JSON
	badMap := map[string]any{
		"fn": func() {}, // Functions can't be marshaled to JSON
	}

	handler := slog.NewTextHandler(os.Stdout, nil)
	result := newEvalResult(handler, badMap, time.Second, "test-id")

	// Should fall back to default string representation
	inspectResult := result.Inspect()
	assert.Contains(t, inspectResult, "map[")
}

// TestExecResult_NestedComplexValues tests the handling of nested complex values
func TestExecResult_NestedComplexValues(t *testing.T) {
	// Create nested complex data structure
	complexValue := map[string]any{
		"string":  "text",
		"number":  42,
		"boolean": true,
		"null":    nil,
		"array":   []any{1, "two", true},
		"map":     map[string]any{"nested": "value"},
	}

	handler := slog.NewTextHandler(os.Stdout, nil)
	result := newEvalResult(handler, complexValue, time.Second, "test-id")

	// Type should be MAP
	assert.Equal(t, data.MAP, result.Type())

	// Inspect should convert to JSON
	inspectResult := result.Inspect()
	require.Contains(t, inspectResult, "string")
	require.Contains(t, inspectResult, "text")
	require.Contains(t, inspectResult, "number")
	require.Contains(t, inspectResult, "42")
	require.Contains(t, inspectResult, "boolean")
	require.Contains(t, inspectResult, "true")
	require.Contains(t, inspectResult, "null")
	require.Contains(t, inspectResult, "array")
	require.Contains(t, inspectResult, "map")
	require.Contains(t, inspectResult, "nested")
	require.Contains(t, inspectResult, "value")

	// Interface should return the original complex structure
	assert.Equal(t, complexValue, result.Interface())
}

// TestExecResult_StringRepresentation tests various string representation cases
func TestExecResult_StringRepresentation(t *testing.T) {
	tests := []struct {
		name            string
		value           any
		valueTypeString string
	}{
		{"nil value", nil, "none"},
		{"string value", "test", "string"},
		{"int value", int32(42), "int"}, // Use int32 instead of int to match implementation
		{"float value", 3.14, "float"},
		{"bool value", true, "bool"},
		{"map value", map[string]any{"key": "value"}, "map"},
		{"list value", []any{1, 2, 3}, "list"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			result := newEvalResult(handler, tt.value, 100*time.Millisecond, "test-123")

			// Check string method
			strResult := result.String()

			// Should contain all essential information
			assert.Contains(t, strResult, "execResult")
			assert.Contains(t, strResult, tt.valueTypeString)
			assert.Contains(t, strResult, "100ms")
			assert.Contains(t, strResult, "test-123")
		})
	}
}
