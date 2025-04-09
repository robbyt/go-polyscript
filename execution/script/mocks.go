package script

import (
	"io"

	machineTypes "github.com/robbyt/go-polyscript/machines/types"
	"github.com/stretchr/testify/mock"
)

// MockCompiler is a mock implementation of the Compiler interface.
type MockCompiler struct {
	mock.Mock
}

// Compile mocks the Compile method of the Compiler interface.
func (m *MockCompiler) Compile(scriptReader io.ReadCloser) (ExecutableContent, error) {
	args := m.Called(scriptReader)
	execContent, ok := args.Get(0).(ExecutableContent)
	if !ok {
		return nil, args.Error(1)
	}
	return execContent, args.Error(1)
}

// MockExecutableContent is a mock implementation of the ExecutableContent interface for testing.
type MockExecutableContent struct {
	mock.Mock
}

func (m *MockExecutableContent) GetSource() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockExecutableContent) GetByteCode() any {
	args := m.Called()
	return args.Get(0)
}

func (m *MockExecutableContent) GetMachineType() machineTypes.Type {
	args := m.Called()
	return args.Get(0).(machineTypes.Type)
}
