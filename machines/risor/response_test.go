package risor

import (
	"log/slog"
	"os"
	"testing"
	"time"

	rObj "github.com/risor-io/risor/object"
	"github.com/risor-io/risor/op"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/data"
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

func TestNewEvalResult(t *testing.T) {
	mockObj := new(RisorObjectMock)

	execTime := 100 * time.Millisecond
	versionID := "test-version-1"

	handler := slog.NewTextHandler(os.Stdout, nil)
	result := newEvalResult(handler, mockObj, execTime, versionID)

	require.NotNil(t, result)
	require.Equal(t, mockObj, result.Object)

	require.Equal(t, execTime, result.execTime)
	assert.Equal(t, execTime.String(), result.GetExecTime())

	require.Equal(t, versionID, result.scriptExeID)
	require.Equal(t, versionID, result.GetScriptExeID())
	require.Implements(t, (*engine.EvaluatorResponse)(nil), result)

	mockObj.AssertExpectations(t)
}

func TestExecResult_Type(t *testing.T) {
	testCases := []struct {
		name     string
		typeStr  string
		expected data.Types
	}{
		{"string type", string(data.STRING), data.STRING},
		{"int type", string(data.INT), data.INT},
		{"bool type", string(data.BOOL), data.BOOL},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockObj := new(RisorObjectMock)
			mockObj.On("Type").Return(rObj.Type(tc.typeStr))

			handler := slog.NewTextHandler(os.Stdout, nil)
			result := newEvalResult(handler, mockObj, time.Second, "version-1")
			assert.Equal(t, tc.expected, result.Type())

			mockObj.AssertExpectations(t)
		})
	}
}

func TestExecResult_String(t *testing.T) {
	testCases := []struct {
		name       string
		mockType   rObj.Type
		mockString string
		execTime   time.Duration
		versionID  string
		expected   string
	}{
		{
			name:       "simple string object",
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockObj := new(RisorObjectMock)
			mockObj.On("Type").Return(tc.mockType)
			mockObj.On("String").Return(tc.mockString)

			handler := slog.NewTextHandler(os.Stdout, nil)
			result := newEvalResult(handler, mockObj, tc.execTime, tc.versionID)
			actual := result.String()
			assert.Equal(t, tc.expected, actual)

			mockObj.AssertExpectations(t)
		})
	}
}
