package compiler

import (
	"testing"

	machineTypes "github.com/robbyt/go-polyscript/engines/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	starlarkLib "go.starlark.net/starlark"
)

// TestExecutable tests the functionality of Executable
func TestExecutable(t *testing.T) {
	t.Parallel()

	// Test creation scenarios
	t.Run("Creation", func(t *testing.T) {
		t.Run("valid creation", func(t *testing.T) {
			content := "print('Hello, World!')"
			bytecode := &starlarkLib.Program{}

			exe := newExecutable([]byte(content), bytecode)
			require.NotNil(t, exe)
			assert.Equal(t, content, exe.GetSource())
			assert.Equal(t, bytecode, exe.GetByteCode())
			assert.Equal(t, bytecode, exe.GetStarlarkByteCode())
			assert.Equal(t, machineTypes.Starlark, exe.GetMachineType())
		})

		t.Run("nil content", func(t *testing.T) {
			bytecode := &starlarkLib.Program{}
			exe := newExecutable(nil, bytecode)
			assert.Nil(t, exe)
		})

		t.Run("nil bytecode", func(t *testing.T) {
			content := "print('test')"
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
		content := "print('Hello, World!')"
		bytecode := &starlarkLib.Program{}
		executable := newExecutable([]byte(content), bytecode)
		require.NotNil(t, executable)

		t.Run("GetSource", func(t *testing.T) {
			source := executable.GetSource()
			assert.Equal(t, content, source)
		})

		t.Run("GetByteCode", func(t *testing.T) {
			code := executable.GetByteCode()
			assert.Equal(t, bytecode, code)

			// Test type assertion
			_, ok := code.(*starlarkLib.Program)
			assert.True(t, ok)
		})

		t.Run("GetStarlarkByteCode", func(t *testing.T) {
			code := executable.GetStarlarkByteCode()
			assert.Equal(t, bytecode, code)
		})

		t.Run("GetMachineType", func(t *testing.T) {
			machineType := executable.GetMachineType()
			assert.Equal(t, machineTypes.Starlark, machineType)
		})
	})
}
