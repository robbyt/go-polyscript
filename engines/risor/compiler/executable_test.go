package compiler

import (
	"testing"

	"github.com/deepnoodle-ai/risor/v2/pkg/bytecode"
	machineTypes "github.com/robbyt/go-polyscript/engines/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecutable tests the functionality of Executable
func TestExecutable(t *testing.T) {
	t.Parallel()

	// Test creation scenarios
	t.Run("Creation", func(t *testing.T) {
		t.Run("valid creation", func(t *testing.T) {
			content := "'Hello, World!'"
			bc := bytecode.NewCode(bytecode.CodeParams{})

			exe := newExecutable([]byte(content), bc)
			require.NotNil(t, exe)
			assert.Equal(t, content, exe.GetSource())
			assert.Equal(t, bc, exe.GetByteCode())
			assert.Equal(t, bc, exe.GetRisorByteCode())
			assert.Equal(t, machineTypes.Risor, exe.GetMachineType())
		})

		t.Run("nil content", func(t *testing.T) {
			bc := bytecode.NewCode(bytecode.CodeParams{})
			exe := newExecutable(nil, bc)
			assert.Nil(t, exe)
		})

		t.Run("nil bytecode", func(t *testing.T) {
			content := "'test'"
			exe := newExecutable([]byte(content), nil)
			assert.Nil(t, exe)
		})

		t.Run("both nil", func(t *testing.T) {
			exe := newExecutable(nil, nil)
			assert.Nil(t, exe)
		})
	})

	// Test getters
	t.Run("Getters", func(t *testing.T) {
		content := "'Hello, World!'"
		bc := bytecode.NewCode(bytecode.CodeParams{})
		executable := newExecutable([]byte(content), bc)
		require.NotNil(t, executable)

		t.Run("GetSource", func(t *testing.T) {
			source := executable.GetSource()
			assert.Equal(t, content, source)
		})

		t.Run("GetByteCode", func(t *testing.T) {
			code := executable.GetByteCode()
			assert.Equal(t, bc, code)

			// Test type assertion
			_, ok := code.(*bytecode.Code)
			assert.True(t, ok)
		})

		t.Run("GetRisorByteCode", func(t *testing.T) {
			code := executable.GetRisorByteCode()
			assert.Equal(t, bc, code)
		})

		t.Run("GetMachineType", func(t *testing.T) {
			machineType := executable.GetMachineType()
			assert.Equal(t, machineTypes.Risor, machineType)
		})
	})
}
