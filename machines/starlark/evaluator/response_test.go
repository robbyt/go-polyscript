package evaluator

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

// StarlarkValueMock is a mock implementation of the starlark.Value interface
type StarlarkValueMock struct {
	mock.Mock
}

func (m *StarlarkValueMock) String() string {
	args := m.Called()
	return args.String(0)
}

func (m *StarlarkValueMock) Type() string {
	args := m.Called()
	return args.String(0)
}

func (m *StarlarkValueMock) Hash() (uint32, error) {
	args := m.Called()
	return args.Get(0).(uint32), args.Error(1)
}

func (m *StarlarkValueMock) Truth() starlark.Bool {
	args := m.Called()
	return args.Get(0).(starlark.Bool)
}

func (m *StarlarkValueMock) Freeze() {
	m.Called()
}

func TestNewEvalResult(t *testing.T) {
	mockVal := new(StarlarkValueMock)
	execTime := 100 * time.Millisecond
	versionID := "test-version-1"

	handler := slog.NewTextHandler(os.Stdout, nil)
	result := newEvalResult(handler, mockVal, execTime, versionID)

	require.NotNil(t, result)
	require.Equal(t, mockVal, result.Value)
	require.Equal(t, execTime, result.execTime)
	assert.Equal(t, execTime.String(), result.GetExecTime())
	require.Equal(t, versionID, result.scriptExeID)
	require.Equal(t, versionID, result.GetScriptExeID())
	require.Implements(t, (*engine.EvaluatorResponse)(nil), result)

	mockVal.AssertExpectations(t)
}

func TestExecResult_Type(t *testing.T) {
	testCases := []struct {
		name     string
		typeStr  string
		expected data.Types
	}{
		{"none type", "NoneType", data.NONE},
		{"string type", "string", data.STRING},
		{"int type", "int", data.INT},
		{"float type", "float", data.FLOAT},
		{"bool type", "bool", data.BOOL},
		{"list type", "list", data.LIST},
		{"tuple type", "tuple", data.TUPLE},
		{"dict type", "dict", data.MAP},
		{"set type", "set", data.SET},
		{"function type", "function", data.FUNCTION},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockVal := new(StarlarkValueMock)
			mockVal.On("Type").Return(tc.typeStr)

			handler := slog.NewTextHandler(os.Stdout, nil)
			result := newEvalResult(handler, mockVal, time.Second, "version-1")
			assert.Equal(t, tc.expected, result.Type())

			mockVal.AssertExpectations(t)
		})
	}
}

func TestExecResult_String(t *testing.T) {
	testCases := []struct {
		name       string
		mockType   string
		mockString string
		execTime   time.Duration
		versionID  string
		expected   string
	}{
		{
			name:       "string value",
			mockType:   "string",
			mockString: "hello",
			execTime:   100 * time.Millisecond,
			versionID:  "v1.0.0",
			expected:   "ExecResult{Type: string, Value: hello, ExecTime: 100ms, ScriptExeID: v1.0.0}",
		},
		{
			name:       "int value",
			mockType:   "int",
			mockString: "42",
			execTime:   200 * time.Millisecond,
			versionID:  "v2.0.0",
			expected:   "ExecResult{Type: int, Value: 42, ExecTime: 200ms, ScriptExeID: v2.0.0}",
		},
		{
			name:       "bool value",
			mockType:   "bool",
			mockString: "True",
			execTime:   50 * time.Millisecond,
			versionID:  "v3.0.0",
			expected:   "ExecResult{Type: bool, Value: True, ExecTime: 50ms, ScriptExeID: v3.0.0}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockVal := new(StarlarkValueMock)
			mockVal.On("Type").Return(tc.mockType)
			mockVal.On("String").Return(tc.mockString)

			handler := slog.NewTextHandler(os.Stdout, nil)
			result := newEvalResult(handler, mockVal, tc.execTime, tc.versionID)
			actual := result.String()
			assert.Equal(t, tc.expected, actual)

			mockVal.AssertExpectations(t)
		})
	}
}
