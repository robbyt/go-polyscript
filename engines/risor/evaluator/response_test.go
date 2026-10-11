package evaluator

import (
	"log/slog"
	"os"
	"testing"
	"time"

	rObj "github.com/deepnoodle-ai/risor/v2/pkg/object"
	"github.com/deepnoodle-ai/risor/v2/pkg/op"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// RisorObjectMock is a mock implementation of the rObj.Object interface.
type RisorObjectMock struct {
	mock.Mock
}

func (m *RisorObjectMock) Type() rObj.Type {
	args := m.Called()
	return args.Get(0).(rObj.Type)
}

func (m *RisorObjectMock) Inspect() string {
	args := m.Called()
	return args.String(0)
}

func (m *RisorObjectMock) Interface() any {
	args := m.Called()
	return args.Get(0)
}

func (m *RisorObjectMock) String() string {
	args := m.Called()
	return args.String(0)
}

func (m *RisorObjectMock) Equals(other rObj.Object) bool {
	args := m.Called(other)
	return args.Bool(0)
}

func (m *RisorObjectMock) Attrs() []rObj.AttrSpec {
	args := m.Called()
	if v := args.Get(0); v != nil {
		return v.([]rObj.AttrSpec)
	}
	return nil
}

func (m *RisorObjectMock) GetAttr(name string) (rObj.Object, bool) {
	args := m.Called(name)
	return args.Get(0).(rObj.Object), args.Bool(1)
}

func (m *RisorObjectMock) SetAttr(name string, value rObj.Object) error {
	args := m.Called(name, value)
	return args.Error(0)
}

func (m *RisorObjectMock) IsTruthy() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *RisorObjectMock) RunOperation(opType op.BinaryOpType, right rObj.Object) (rObj.Object, error) {
	args := m.Called(opType, right)
	return args.Get(0).(rObj.Object), args.Error(1)
}

// TestResponseMethods tests all the methods of the EvaluatorResponse interface
func TestResponseMethods(t *testing.T) {
	t.Parallel()

	t.Run("Creation", func(t *testing.T) {
		tests := []struct {
			name      string
			setupMock func() *RisorObjectMock
			execTime  time.Duration
			versionID string
		}{
			{
				name: "with valid object",
				setupMock: func() *RisorObjectMock {
					mockObj := new(RisorObjectMock)
					return mockObj
				},
				execTime:  100 * time.Millisecond,
				versionID: "test-version-1",
			},
			{
				name: "with longer execution time",
				setupMock: func() *RisorObjectMock {
					mockObj := new(RisorObjectMock)
					return mockObj
				},
				execTime:  2 * time.Second,
				versionID: "test-version-2",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockObj := tt.setupMock()
				handler := slog.NewTextHandler(os.Stdout, nil)

				result := newEvalResult(handler, mockObj, tt.execTime, tt.versionID)

				// Verify basic properties
				require.NotNil(t, result)
				require.Equal(t, mockObj, result.Object)
				require.Equal(t, tt.execTime, result.execTime)
				require.Equal(t, tt.versionID, result.scriptExeID)

				// Verify interface implementation
				require.Implements(t, (*platform.EvaluatorResponse)(nil), result)

				// Verify metadata methods
				assert.Equal(t, tt.execTime, result.ExecTime())
				assert.Equal(t, tt.versionID, result.ScriptExeID())
			})
		}
	})

	t.Run("Type", func(t *testing.T) {
		tests := []struct {
			name     string
			typeStr  rObj.Type
			expected data.Types
		}{
			{"nil type", rObj.NIL, data.NONE},
			{"bool type", rObj.BOOL, data.BOOL},
			{"int type", rObj.INT, data.INT},
			{"byte type", rObj.BYTE, data.INT},
			{"float type", rObj.FLOAT, data.FLOAT},
			{"string type", rObj.STRING, data.STRING},
			{"bytes type", rObj.BYTES, data.BYTES},
			{"list type", rObj.LIST, data.LIST},
			{"range type", rObj.RANGE, data.LIST},
			{"iter type", rObj.ITER, data.LIST},
			{"map type", rObj.MAP, data.MAP},
			{"function type", rObj.FUNCTION, data.FUNCTION},
			{"builtin type", rObj.BUILTIN, data.FUNCTION},
			{"go_func type", rObj.GOFUNC, data.FUNCTION},
			{"partial type", rObj.PARTIAL, data.FUNCTION},
			{"error type", rObj.ERROR, data.ERROR},
			{"time type falls back to error", rObj.TIME, data.ERROR},
			{"module type falls back to error", rObj.MODULE, data.ERROR},
			{"go_struct type falls back to error", rObj.GOSTRUCT, data.ERROR},
			{"module-defined type falls back to error", rObj.Type("regexp"), data.ERROR},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockObj := new(RisorObjectMock)
				mockObj.On("Type").Return(tt.typeStr)

				handler := slog.NewTextHandler(os.Stdout, nil)
				result := newEvalResult(handler, mockObj, time.Second, "version-1")

				// Check the result type
				assert.Equal(t, tt.expected, result.Type())

				// Verify mock expectations
				mockObj.AssertExpectations(t)
			})
		}

		t.Run("nil object", func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, nil)
			result := newEvalResult(handler, nil, time.Second, "version-1")
			assert.Equal(t, data.NONE, result.Type())
			assert.Nil(t, result.Interface())
		})
	})

	t.Run("Type and Interface agree", func(t *testing.T) {
		now := time.Now()
		tests := []struct {
			name          string
			obj           rObj.Object
			expectedType  data.Types
			expectedValue any
		}{
			{"nil", rObj.Nil, data.NONE, nil},
			{"int", rObj.NewInt(42), data.INT, int64(42)},
			{"byte", rObj.NewByte(65), data.INT, byte(65)},
			{"float", rObj.NewFloat(1.5), data.FLOAT, 1.5},
			{"string", rObj.NewString("s"), data.STRING, "s"},
			{"bool", rObj.True, data.BOOL, true},
			{"bytes", rObj.NewBytes([]byte("abc")), data.BYTES, []byte("abc")},
			{
				"list",
				rObj.NewList([]rObj.Object{rObj.NewInt(1), rObj.Nil}),
				data.LIST,
				[]any{int64(1), nil},
			},
			{"range", rObj.NewRange(0, 3, 1), data.LIST, []any{int64(0), int64(1), int64(2)}},
			{"empty range", rObj.NewRange(0, 0, 1), data.LIST, []any{}},
			{
				"map",
				rObj.NewMap(map[string]rObj.Object{"a": rObj.NewInt(1)}),
				data.MAP,
				map[string]any{"a": int64(1)},
			},
			{"builtin", rObj.NewNoopBuiltin("noop"), data.FUNCTION, nil},
			{"time falls back to error", rObj.NewTime(now), data.ERROR, now},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				handler := slog.NewTextHandler(os.Stdout, nil)
				result := newEvalResult(handler, tt.obj, time.Second, "version-1")
				assert.Equal(t, tt.expectedType, result.Type(), "Type() should classify the object")
				assert.Equal(t, tt.expectedValue, result.Interface(), "Interface() should return the converted object")
			})
		}
	})

	t.Run("String", func(t *testing.T) {
		tests := []struct {
			name       string
			mockType   rObj.Type
			mockString string
			execTime   time.Duration
			versionID  string
			expected   string
		}{
			{
				name:       "string object",
				mockType:   rObj.Type("string"),
				mockString: "hello",
				execTime:   100 * time.Millisecond,
				versionID:  "v1.0.0",
				expected:   "ExecResult{Type: string, Value: hello, ExecTime: 100ms, ScriptExeID: v1.0.0}",
			},
			{
				name:       "integer object",
				mockType:   rObj.INT,
				mockString: "42",
				execTime:   200 * time.Millisecond,
				versionID:  "v2.0.0",
				expected:   "ExecResult{Type: int, Value: 42, ExecTime: 200ms, ScriptExeID: v2.0.0}",
			},
			{
				name:       "boolean object",
				mockType:   rObj.BOOL,
				mockString: "true",
				execTime:   50 * time.Millisecond,
				versionID:  "v3.0.0",
				expected:   "ExecResult{Type: bool, Value: true, ExecTime: 50ms, ScriptExeID: v3.0.0}",
			},
			{
				name:       "nil object",
				mockType:   rObj.NIL,
				mockString: "nil",
				execTime:   10 * time.Millisecond,
				versionID:  "v4.0.0",
				expected:   "ExecResult{Type: none, Value: nil, ExecTime: 10ms, ScriptExeID: v4.0.0}",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockObj := new(RisorObjectMock)
				mockObj.On("Type").Return(tt.mockType)
				mockObj.On("String").Return(tt.mockString)

				handler := slog.NewTextHandler(os.Stdout, nil)
				result := newEvalResult(handler, mockObj, tt.execTime, tt.versionID)

				// Check string representation
				actual := result.String()
				assert.Equal(t, tt.expected, actual)

				// Verify mock expectations
				mockObj.AssertExpectations(t)
			})
		}
	})

	t.Run("Inspect", func(t *testing.T) {
		tests := []struct {
			name            string
			mockInspect     string
			expectedInspect string
		}{
			{
				name:            "string value",
				mockInspect:     "\"test string\"",
				expectedInspect: "\"test string\"",
			},
			{
				name:            "number value",
				mockInspect:     "42",
				expectedInspect: "42",
			},
			{
				name:            "boolean value",
				mockInspect:     "true",
				expectedInspect: "true",
			},
			{
				name:            "complex value",
				mockInspect:     "{\"key\":\"value\"}",
				expectedInspect: "{\"key\":\"value\"}",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockObj := new(RisorObjectMock)
				mockObj.On("Inspect").Return(tt.mockInspect)

				handler := slog.NewTextHandler(os.Stdout, nil)
				result := newEvalResult(handler, mockObj, time.Second, "test-1")

				// Check inspect result
				assert.Equal(t, tt.expectedInspect, result.Inspect())

				// Verify mock expectations
				mockObj.AssertExpectations(t)
			})
		}
	})

	t.Run("Interface", func(t *testing.T) {
		tests := []struct {
			name      string
			mockValue any
		}{
			{
				name:      "string value",
				mockValue: "test string",
			},
			{
				name:      "number value",
				mockValue: 42,
			},
			{
				name:      "boolean value",
				mockValue: true,
			},
			{
				name:      "map value",
				mockValue: map[string]any{"key": "value"},
			},
			{
				name:      "nil value",
				mockValue: nil,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockObj := new(RisorObjectMock)
				mockObj.On("Interface").Return(tt.mockValue)

				handler := slog.NewTextHandler(os.Stdout, nil)
				result := newEvalResult(handler, mockObj, time.Second, "test-1")

				// The Interface method should return the original value
				actual := result.Interface()
				assert.Equal(t, tt.mockValue, actual)

				// Verify mock expectations
				mockObj.AssertExpectations(t)
			})
		}
	})

	t.Run("NilHandler", func(t *testing.T) {
		mockObj := new(RisorObjectMock)
		execTime := 100 * time.Millisecond
		versionID := "test-version-1"

		// Create with nil handler
		result := newEvalResult(nil, mockObj, execTime, versionID)

		// Should create default handler and logger
		require.NotNil(t, result)
		require.NotNil(t, result.logHandler)
		require.NotNil(t, result.logger)

		// Should still store all values correctly
		assert.Equal(t, mockObj, result.Object)
		assert.Equal(t, execTime, result.execTime)
		assert.Equal(t, versionID, result.scriptExeID)
	})

	t.Run("Metadata", func(t *testing.T) {
		tests := []struct {
			name      string
			execTime  time.Duration
			versionID string
		}{
			{
				name:      "short execution time",
				execTime:  123 * time.Millisecond,
				versionID: "test-script-9876",
			},
			{
				name:      "long execution time",
				execTime:  3 * time.Second,
				versionID: "test-script-1234",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockObj := new(RisorObjectMock)
				handler := slog.NewTextHandler(os.Stdout, nil)

				result := newEvalResult(handler, mockObj, tt.execTime, tt.versionID)

				// Test ScriptExeID
				assert.Equal(t, tt.versionID, result.ScriptExeID())

				// Test ExecTime
				assert.Equal(t, tt.execTime, result.ExecTime())
			})
		}
	})
}

func TestExecResultAsMap(t *testing.T) {
	t.Parallel()
	handler := slog.NewTextHandler(os.Stdout, nil)

	t.Run("success", func(t *testing.T) {
		m := map[string]any{"k": "v", "n": 42}
		mockObj := new(RisorObjectMock)
		mockObj.On("Interface").Return(m)
		result := newEvalResult(handler, mockObj, time.Millisecond, "id-ok")
		got, err := result.AsMap()
		require.NoError(t, err)
		assert.Equal(t, m, got)
	})

	t.Run("error on string", func(t *testing.T) {
		mockObj := new(RisorObjectMock)
		mockObj.On("Interface").Return("not a map")
		result := newEvalResult(handler, mockObj, time.Millisecond, "id-bad")
		got, err := result.AsMap()
		require.Error(t, err)
		assert.Nil(t, got)
		require.ErrorIs(t, err, ErrAsMapTypeMismatch)
		assert.Contains(t, err.Error(), "got string")
	})

	t.Run("error on int", func(t *testing.T) {
		mockObj := new(RisorObjectMock)
		mockObj.On("Interface").Return(42)
		result := newEvalResult(handler, mockObj, time.Millisecond, "id-int")
		got, err := result.AsMap()
		require.Error(t, err)
		assert.Nil(t, got)
		require.ErrorIs(t, err, ErrAsMapTypeMismatch)
		assert.Contains(t, err.Error(), "got int")
	})
}
