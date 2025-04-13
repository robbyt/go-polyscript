package script

import (
	"testing"

	machineTypes "github.com/robbyt/go-polyscript/engines/types"
	"github.com/stretchr/testify/require"
)

func TestExecutableContent(t *testing.T) {
	t.Parallel()

	t.Run("GetSource", func(t *testing.T) {
		mockContent := new(MockExecutableContent)
		expectedSource := "print('Hello, World!')"
		mockContent.On("GetSource").Return(expectedSource)

		source := mockContent.GetSource()
		require.Equal(t, expectedSource, source, "Expected source to match")
		mockContent.AssertExpectations(t)
	})

	t.Run("GetByteCode", func(t *testing.T) {
		mockContent := new(MockExecutableContent)
		expectedByteCode := []byte{0x01, 0x02, 0x03}
		mockContent.On("GetByteCode").Return(expectedByteCode)

		byteCode := mockContent.GetByteCode()
		require.Equal(t, expectedByteCode, byteCode, "Expected bytecode to match")
		mockContent.AssertExpectations(t)
	})

	t.Run("GetMachineType", func(t *testing.T) {
		mockContent := new(MockExecutableContent)
		expectedType := machineTypes.Risor
		mockContent.On("GetMachineType").Return(expectedType)

		machineType := mockContent.GetMachineType()
		require.Equal(t, expectedType, machineType, "Expected machine type to match")
		mockContent.AssertExpectations(t)
	})
}
