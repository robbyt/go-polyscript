package evaluator

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/robbyt/go-polyscript/abstract/data"
	"github.com/robbyt/go-polyscript/abstract/evaluation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResponseMethods tests all methods of the EvaluatorResponse interface
func TestResponseMethods(t *testing.T) {
	t.Parallel()

	t.Run("Creation", func(t *testing.T) {
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
			t.Run(tt.name, func(t *testing.T) {
				handler := slog.NewTextHandler(os.Stdout, nil)
				result := newEvalResult(handler, tt.value, tt.execTime, tt.versionID)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectValue, result.value)
				assert.Equal(t, tt.execTime, result.execTime)
				assert.Equal(t, tt.versionID, result.scriptExeID)
				require.Implements(t, (*evaluation.EvaluatorResponse)(nil), result)
			})
		}
	})

	t.Run("Type", func(t *testing.T) {
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
			t.Run(tt.name, func(t *testing.T) {
				handler := slog.NewTextHandler(os.Stdout, nil)
				result := newEvalResult(handler, tt.value, time.Second, "test-1")
				assert.Equal(t, tt.expected, result.Type())
			})
		}

		t.Run("unknown type", func(t *testing.T) {
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
		})
	})

	t.Run("String", func(t *testing.T) {
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
			t.Run(tt.name, func(t *testing.T) {
				handler := slog.NewTextHandler(os.Stdout, nil)
				result := newEvalResult(handler, tt.value, tt.execTime, tt.versionID)
				assert.Equal(t, tt.expected, result.String())
			})
		}

		t.Run("string representation coverage", func(t *testing.T) {
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
		})
	})

	t.Run("Inspect", func(t *testing.T) {
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
			t.Run(tt.name, func(t *testing.T) {
				handler := slog.NewTextHandler(os.Stdout, nil)
				result := newEvalResult(handler, tt.value, time.Second, "test-1")
				assert.Equal(t, tt.expected, result.Inspect())
			})
		}

		t.Run("with invalid JSON", func(t *testing.T) {
			// Create a map with a value that can't be marshaled to JSON
			badMap := map[string]any{
				"fn": func() {}, // Functions can't be marshaled to JSON
			}

			handler := slog.NewTextHandler(os.Stdout, nil)
			result := newEvalResult(handler, badMap, time.Second, "test-id")

			// Should fall back to default string representation
			inspectResult := result.Inspect()
			assert.Contains(t, inspectResult, "map[")
		})

		t.Run("nested complex values", func(t *testing.T) {
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
		})
	})

	t.Run("Interface", func(t *testing.T) {
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
	})

	t.Run("Metadata", func(t *testing.T) {
		tests := []struct {
			name      string
			value     any
			execTime  time.Duration
			versionID string
		}{
			{
				name:      "short execution time",
				value:     "test string",
				execTime:  123 * time.Millisecond,
				versionID: "test-script-9876",
			},
			{
				name:      "long execution time",
				value:     42,
				execTime:  3 * time.Second,
				versionID: "test-script-1234",
			},
			{
				name:      "microsecond execution time",
				value:     true,
				execTime:  500 * time.Microsecond,
				versionID: "test-script-5678",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				handler := slog.NewTextHandler(os.Stdout, nil)
				result := newEvalResult(handler, tt.value, tt.execTime, tt.versionID)

				// Test GetScriptExeID
				assert.Equal(t, tt.versionID, result.GetScriptExeID())

				// Test GetExecTime
				assert.Equal(t, tt.execTime.String(), result.GetExecTime())
			})
		}
	})

	t.Run("NilHandler", func(t *testing.T) {
		tests := []struct {
			name      string
			value     any
			execTime  time.Duration
			versionID string
		}{
			{
				name:      "string value",
				value:     "test value",
				execTime:  100 * time.Millisecond,
				versionID: "test-id",
			},
			{
				name:      "numeric value",
				value:     42,
				execTime:  2 * time.Second,
				versionID: "numeric-test-id",
			},
			{
				name:      "boolean value",
				value:     true,
				execTime:  50 * time.Millisecond,
				versionID: "boolean-test-id",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create with nil handler
				result := newEvalResult(nil, tt.value, tt.execTime, tt.versionID)

				// Should create default handler and logger
				require.NotNil(t, result)
				require.NotNil(t, result.logHandler)
				require.NotNil(t, result.logger)

				// Should still store all values correctly
				assert.Equal(t, tt.value, result.value)
				assert.Equal(t, tt.execTime, result.execTime)
				assert.Equal(t, tt.versionID, result.scriptExeID)
			})
		}
	})
}
