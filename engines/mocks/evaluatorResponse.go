package mocks

import (
	"github.com/robbyt/go-polyscript/abstract/data"
	"github.com/stretchr/testify/mock"
)

// EvaluatorResponse is a mock implementation of the cpu.EvaluatorResponse interface.
type EvaluatorResponse struct {
	mock.Mock
}

// Type returns a mockable Type.
func (m *EvaluatorResponse) Type() data.Types {
	args := m.Called()
	val := args.Get(0)

	switch val.(type) {
	case bool:
		return data.BOOL
	case int:
		return data.INT
	case map[string]any, map[any]any:
		return data.MAP
	case string:
		return data.STRING
	default:
		// If the mock was set up with a data.Types directly, return it
		if t, ok := val.(data.Types); ok {
			return t
		}
		panic("unknown type")
	}
}

// Inspect returns a mockable string.
func (m *EvaluatorResponse) Inspect() string {
	args := m.Called()
	return args.String(0)
}

// Interface returns a mockable value of "any" type, and must be type asserted to the correct type.
func (m *EvaluatorResponse) Interface() any {
	args := m.Called()
	return args.Get(0)
}

// GetScriptExeID returns a mockable script version.
func (m *EvaluatorResponse) GetScriptExeID() string {
	args := m.Called()
	return args.String(0)
}

// GetExecTime returns a mockable execution time.
func (m *EvaluatorResponse) GetExecTime() string {
	args := m.Called()
	return args.String(0)
}
