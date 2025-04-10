package evaluator

import (
	"log/slog"
	"os"
	"testing"
	"time"

	rObj "github.com/risor-io/risor/object"
	"github.com/risor-io/risor/op"
	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/data"
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

func (m *RisorObjectMock) Hash() (uint32, error) {
	args := m.Called()
	return args.Get(0).(uint32), args.Error(1)
}

func (m *RisorObjectMock) String() string {
	args := m.Called()
	return args.String(0)
}

func (m *RisorObjectMock) Cost() int {
	args := m.Called()
	return args.Int(0)
}

func (m *RisorObjectMock) Equals(other rObj.Object) rObj.Object {
	args := m.Called(other)
	return args.Get(0).(rObj.Object)
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

func (m *RisorObjectMock) RunOperation(opType op.BinaryOpType, right rObj.Object) rObj.Object {
	args := m.Called(opType, right)
	return args.Get(0).(rObj.Object)
}

func (m *RisorObjectMock) Compare(other rObj.Object) (int, error) {
	args := m.Called(other)
	return args.Int(0), args.Error(1)
}

// TestEvalResult_Creation tests creating a new evaluation result
func TestEvalResult_Creation(t *testing.T) {
	t.Parallel()

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
			require.Implements(t, (*engine.EvaluatorResponse)(nil), result)

			// Verify metadata methods
			assert.Equal(t, tt.execTime.String(), result.GetExecTime())
			assert.Equal(t, tt.versionID, result.GetScriptExeID())
		})
	}
}

// TestEvalResult_Type tests the Type method
func TestEvalResult_Type(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typeStr  string
		expected data.Types
	}{
		{"string type", string(data.STRING), data.STRING},
		{"int type", string(data.INT), data.INT},
		{"bool type", string(data.BOOL), data.BOOL},
		{"float type", string(data.FLOAT), data.FLOAT},
		{"list type", string(data.LIST), data.LIST},
		{"map type", string(data.MAP), data.MAP},
		{"none type", string(data.NONE), data.NONE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockObj := new(RisorObjectMock)
			mockObj.On("Type").Return(rObj.Type(tt.typeStr))

			handler := slog.NewTextHandler(os.Stdout, nil)
			result := newEvalResult(handler, mockObj, time.Second, "version-1")

			// Check the result type
			assert.Equal(t, tt.expected, result.Type())

			// Verify mock expectations
			mockObj.AssertExpectations(t)
		})
	}
}

// TestEvalResult_String tests the String method
func TestEvalResult_String(t *testing.T) {
	t.Parallel()

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
			mockType:   rObj.Type("integer"),
			mockString: "42",
			execTime:   200 * time.Millisecond,
			versionID:  "v2.0.0",
			expected:   "ExecResult{Type: integer, Value: 42, ExecTime: 200ms, ScriptExeID: v2.0.0}",
		},
		{
			name:       "boolean object",
			mockType:   rObj.Type("boolean"),
			mockString: "true",
			execTime:   50 * time.Millisecond,
			versionID:  "v3.0.0",
			expected:   "ExecResult{Type: boolean, Value: true, ExecTime: 50ms, ScriptExeID: v3.0.0}",
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
}

// TestEvalResult_Inspect tests the Inspect method
func TestEvalResult_Inspect(t *testing.T) {
	t.Parallel()

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
}

// TestEvalResult_Interface tests the Interface method
func TestEvalResult_Interface(t *testing.T) {
	t.Parallel()

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
}

// TestEvalResult_NilHandler tests creating a result with nil handler
func TestEvalResult_NilHandler(t *testing.T) {
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
}

// TestEvalResult_Metadata tests all metadata accessors
func TestEvalResult_Metadata(t *testing.T) {
	t.Parallel()

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

			// Test GetScriptExeID
			assert.Equal(t, tt.versionID, result.GetScriptExeID())

			// Test GetExecTime
			assert.Equal(t, tt.execTime.String(), result.GetExecTime())
		})
	}
}
