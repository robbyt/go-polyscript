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
