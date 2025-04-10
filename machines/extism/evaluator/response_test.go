package evaluator

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvalResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		value       any
		execTime    time.Duration
		versionID   string
		expectValue any
	}{
		{
			name:        "string value",
			value:       "hello",
			execTime:    100 * time.Millisecond,
			versionID:   "test-1",
			expectValue: "hello",
		},
		{
			name:        "int value",
			value:       42,
			execTime:    200 * time.Millisecond,
			versionID:   "test-2",
			expectValue: 42,
		},
		{
			name:        "bool value",
			value:       true,
			execTime:    50 * time.Millisecond,
			versionID:   "test-3",
			expectValue: true,
		},
		{
			name:        "nil value",
			value:       nil,
			execTime:    75 * time.Millisecond,
			versionID:   "test-4",
			expectValue: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := slog.NewTextHandler(os.Stdout, nil)
			result := newEvalResult(handler, tt.value, tt.execTime, tt.versionID)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectValue, result.value)
			assert.Equal(t, tt.execTime, result.execTime)
			assert.Equal(t, tt.versionID, result.scriptExeID)
			require.Implements(t, (*engine.EvaluatorResponse)(nil), result)
		})
	}
}

func TestExecResult_Type(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		value    any
		expected data.Types
	}{
		{"nil value", nil, data.NONE},
		{"bool value", true, data.BOOL},
		{"int32 value", int32(42), data.INT},
		{"int64 value", int64(42), data.INT},
		{"uint32 value", uint32(42), data.INT},
		{"uint64 value", uint64(42), data.INT},
		{"float32 value", float32(3.14), data.FLOAT},
		{"float64 value", float64(3.14), data.FLOAT},
		{"string value", "hello", data.STRING},
		{"empty list", []any{}, data.LIST},
		{"list value", []any{1, 2, 3}, data.LIST},
		{"empty dict", map[string]any{}, data.MAP},
		{"dict value", map[string]any{"key": "value"}, data.MAP},
		{"complex dict", map[string]any{
			"str":   "value",
			"num":   42,
			"bool":  true,
			"list":  []any{1, 2, 3},
			"inner": map[string]any{"key": "value"},
		}, data.MAP},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := slog.NewTextHandler(os.Stdout, nil)
			result := newEvalResult(handler, tt.value, time.Second, "test-1")
			assert.Equal(t, tt.expected, result.Type())
		})
	}
}

func TestExecResult_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		value     any
		execTime  time.Duration
		versionID string
		expected  string
	}{
		{
			name:      "string value",
			value:     "hello",
			execTime:  100 * time.Millisecond,
			versionID: "v1.0.0",
			expected:  "execResult{Type: string, Value: hello, ExecTime: 100ms, ScriptExeID: v1.0.0}",
		},
		{
			name:      "int32 value",
			value:     int32(42),
			execTime:  200 * time.Millisecond,
			versionID: "v2.0.0",
			expected:  "execResult{Type: int, Value: 42, ExecTime: 200ms, ScriptExeID: v2.0.0}",
		},
		{
			name:      "float64 value",
			value:     float64(3.14),
			execTime:  300 * time.Millisecond,
			versionID: "v3.0.0",
			expected:  "execResult{Type: float, Value: 3.14, ExecTime: 300ms, ScriptExeID: v3.0.0}",
		},
		{
			name:      "nil value",
			value:     nil,
			execTime:  50 * time.Millisecond,
			versionID: "v4.0.0",
			expected:  "execResult{Type: none, Value: <nil>, ExecTime: 50ms, ScriptExeID: v4.0.0}",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := slog.NewTextHandler(os.Stdout, nil)
			result := newEvalResult(handler, tt.value, tt.execTime, tt.versionID)
			assert.Equal(t, tt.expected, result.String())
		})
	}
}

func TestExecResult_Inspect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{"string value", "hello", "hello"},
		{"int value", 42, "42"},
		{"bool value", true, "true"},
		{"nil value", nil, "<nil>"},
		{"float value", 3.14159, "3.14159"},
		{"list value", []any{1, 2, 3}, "[1 2 3]"},
		{"dict value", map[string]any{"key": "value"}, "{\"key\":\"value\"}"},
		{"complex dict", map[string]any{
			"num":  42,
			"str":  "test",
			"bool": true,
		}, "{\"bool\":true,\"num\":42,\"str\":\"test\"}"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := slog.NewTextHandler(os.Stdout, nil)
			result := newEvalResult(handler, tt.value, time.Second, "test-1")
			assert.Equal(t, tt.expected, result.Inspect())
		})
	}
}

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
