package evaluator

import (
	"log/slog"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/data"
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

// TestResponseMethods tests all the methods of the EvaluatorResponse interface
func TestResponseMethods(t *testing.T) {
	t.Parallel()

	t.Run("Creation", func(t *testing.T) {
		tests := []struct {
			name      string
			execTime  time.Duration
			versionID string
		}{
			{
				name:      "with standard values",
				execTime:  100 * time.Millisecond,
				versionID: "test-version-1",
			},
			{
				name:      "with longer execution time",
				execTime:  5 * time.Second,
				versionID: "test-version-2",
			},
			{
				name:      "with microsecond execution time",
				execTime:  750 * time.Microsecond,
				versionID: "test-version-micro",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockVal := new(StarlarkValueMock)
				handler := slog.NewTextHandler(os.Stdout, nil)

				result := newEvalResult(handler, mockVal, tt.execTime, tt.versionID)

				require.NotNil(t, result)
				require.Equal(t, mockVal, result.Value)
				require.Equal(t, tt.execTime, result.execTime)
				assert.Equal(t, tt.execTime, result.ExecTime())
				require.Equal(t, tt.versionID, result.scriptExeID)
				require.Equal(t, tt.versionID, result.ScriptExeID())
				require.Implements(t, (*platform.EvaluatorResponse)(nil), result)

				mockVal.AssertExpectations(t)
			})
		}
	})

	t.Run("Type", func(t *testing.T) {
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
			{"builtin type", "builtin_function_or_method", data.FUNCTION},
			{"bytes type", "bytes", data.BYTES},
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
	})

	t.Run("Type and Interface agree", func(t *testing.T) {
		bigInt := new(big.Int).Lsh(big.NewInt(1), 64)
		set := starlark.NewSet(2)
		require.NoError(t, set.Insert(starlark.MakeInt(1)))
		require.NoError(t, set.Insert(starlark.MakeInt(2)))

		testCases := []struct {
			name          string
			value         starlark.Value
			expectedType  data.Types
			expectedValue any
		}{
			{"none", starlark.None, data.NONE, nil},
			{"small int", starlark.MakeInt(42), data.INT, int64(42)},
			{"big int", starlark.MakeBigInt(bigInt), data.INT, bigInt},
			{"tuple", starlark.Tuple{starlark.MakeInt(1), starlark.String("a")}, data.TUPLE, []any{int64(1), "a"}},
			{"set", set, data.SET, []any{int64(1), int64(2)}},
			{"bytes", starlark.Bytes("abc"), data.BYTES, []byte("abc")},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				handler := slog.NewTextHandler(os.Stdout, nil)
				result := newEvalResult(handler, tc.value, time.Second, "version-1")
				assert.Equal(t, tc.expectedType, result.Type(), "Type() should classify the value")
				assert.Equal(t, tc.expectedValue, result.Interface(), "Interface() should convert the value without loss")
			})
		}
	})

	t.Run("String", func(t *testing.T) {
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
	})

	t.Run("Inspect", func(t *testing.T) {
		testCases := []struct {
			name            string
			mockStringVal   string
			expectedInspect string
		}{
			{
				name:            "string value",
				mockStringVal:   "\"test string\"",
				expectedInspect: "\"test string\"",
			},
			{
				name:            "number value",
				mockStringVal:   "42",
				expectedInspect: "42",
			},
			{
				name:            "boolean value",
				mockStringVal:   "True",
				expectedInspect: "True",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockVal := new(StarlarkValueMock)
				mockVal.On("String").Return(tc.mockStringVal)

				handler := slog.NewTextHandler(os.Stdout, nil)
				result := newEvalResult(handler, mockVal, time.Second, "test-1")

				assert.Equal(t, tc.expectedInspect, result.Inspect())
				mockVal.AssertExpectations(t)
			})
		}
	})

	t.Run("Metadata", func(t *testing.T) {
		tests := []struct {
			name     string
			execTime time.Duration
			scriptID string
		}{
			{
				name:     "short execution time",
				execTime: 123 * time.Millisecond,
				scriptID: "test-script-9876",
			},
			{
				name:     "long execution time",
				execTime: 3 * time.Second,
				scriptID: "test-script-1234",
			},
			{
				name:     "zero execution time",
				execTime: 0,
				scriptID: "test-script-zero",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockVal := new(StarlarkValueMock)
				handler := slog.NewTextHandler(os.Stdout, nil)
				result := newEvalResult(handler, mockVal, tt.execTime, tt.scriptID)

				// Test ScriptExeID
				assert.Equal(t, tt.scriptID, result.ScriptExeID())

				// Test ExecTime
				assert.Equal(t, tt.execTime, result.ExecTime())
			})
		}
	})

	t.Run("NilHandler", func(t *testing.T) {
		tests := []struct {
			name      string
			execTime  time.Duration
			versionID string
		}{
			{
				name:      "standard case",
				execTime:  100 * time.Millisecond,
				versionID: "test-version-1",
			},
			{
				name:      "long execution time",
				execTime:  3 * time.Second,
				versionID: "test-version-2",
			},
			{
				name:      "very short execution time",
				execTime:  5 * time.Microsecond,
				versionID: "test-version-micro",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockVal := new(StarlarkValueMock)

				// Create with nil handler
				result := newEvalResult(nil, mockVal, tt.execTime, tt.versionID)

				// Should create default handler and logger
				require.NotNil(t, result)
				require.NotNil(t, result.logHandler)
				require.NotNil(t, result.logger)

				// Should still store all values correctly
				assert.Equal(t, mockVal, result.Value)
				assert.Equal(t, tt.execTime, result.execTime)
				assert.Equal(t, tt.versionID, result.scriptExeID)
			})
		}
	})
}

func TestExecResultAsMap(t *testing.T) {
	t.Parallel()
	handler := slog.NewTextHandler(os.Stdout, nil)

	t.Run("success", func(t *testing.T) {
		dict := starlark.NewDict(2)
		require.NoError(t, dict.SetKey(starlark.String("greeting"), starlark.String("hi")))
		require.NoError(t, dict.SetKey(starlark.String("n"), starlark.MakeInt(42)))

		result := newEvalResult(handler, dict, time.Millisecond, "id-ok")
		got, err := result.AsMap()
		require.NoError(t, err)
		assert.Equal(t, "hi", got["greeting"])
		assert.Equal(t, int64(42), got["n"])
	})

	t.Run("error on string", func(t *testing.T) {
		result := newEvalResult(handler, starlark.String("not a map"), time.Millisecond, "id-bad")
		got, err := result.AsMap()
		require.Error(t, err)
		assert.Nil(t, got)
		require.ErrorIs(t, err, ErrAsMapTypeMismatch)
		assert.Contains(t, err.Error(), "got string")
	})

	t.Run("error on int", func(t *testing.T) {
		result := newEvalResult(handler, starlark.MakeInt(123), time.Millisecond, "id-int")
		got, err := result.AsMap()
		require.Error(t, err)
		assert.Nil(t, got)
		require.ErrorIs(t, err, ErrAsMapTypeMismatch)
		// starlark int converts to int64; the AsMap error message reflects the Go type.
		assert.Contains(t, err.Error(), "got int64")
	})
}
