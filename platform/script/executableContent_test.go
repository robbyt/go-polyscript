package script

import (
	"testing"

	engineTypes "github.com/robbyt/go-polyscript/engines/types"
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

	t.Run("EngineType", func(t *testing.T) {
		mockContent := new(MockExecutableContent)
		expectedType := engineTypes.Risor
		mockContent.On("EngineType").Return(expectedType)

		engineType := mockContent.EngineType()
		require.Equal(t, expectedType, engineType, "Expected engine type to match")
		mockContent.AssertExpectations(t)
	})
}
