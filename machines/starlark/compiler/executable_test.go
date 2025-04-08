package compiler

import (
	"testing"

	machineTypes "github.com/robbyt/go-polyscript/machines/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	starlarkLib "go.starlark.net/starlark"
)

func TestNewExecutableValid(t *testing.T) {
	content := "print('Hello, World!')"
	bytecode := &starlarkLib.Program{}

	executable := newExecutable([]byte(content), bytecode)
	require.NotNil(t, executable)
	assert.Equal(t, content, executable.GetSource())
	assert.Equal(t, bytecode, executable.GetByteCode())
	assert.Equal(t, bytecode, executable.GetStarlarkByteCode())
	assert.Equal(t, machineTypes.Starlark, executable.GetMachineType())
}

func TestNewExecutableNilContent(t *testing.T) {
	bytecode := &starlarkLib.Program{}
	executable := newExecutable(nil, bytecode)
	require.Nil(t, executable)
}

func TestNewExecutableNilByteCode(t *testing.T) {
	content := "print('Hello, World!')"
	executable := newExecutable([]byte(content), nil)
	require.Nil(t, executable)
}

func TestNewExecutableNilContentAndByteCode(t *testing.T) {
	executable := newExecutable(nil, nil)
	require.Nil(t, executable)
}

func TestExecutable_GetSource(t *testing.T) {
	content := "print('Hello, World!')"
	bytecode := &starlarkLib.Program{}
	executable := newExecutable([]byte(content), bytecode)
	require.NotNil(t, executable)

	source := executable.GetSource()
	assert.Equal(t, content, source)
}

func TestExecutable_GetByteCode(t *testing.T) {
	content := "print('Hello, World!')"
	bytecode := &starlarkLib.Program{}
	executable := newExecutable([]byte(content), bytecode)
	require.NotNil(t, executable)

	code := executable.GetByteCode()
	assert.Equal(t, bytecode, code)

	// Test type assertion
	_, ok := code.(*starlarkLib.Program)
	assert.True(t, ok)
}

func TestExecutable_GetStarlarkByteCode(t *testing.T) {
	content := "print('Hello, World!')"
	bytecode := &starlarkLib.Program{}
	executable := newExecutable([]byte(content), bytecode)
	require.NotNil(t, executable)

	code := executable.GetStarlarkByteCode()
	assert.Equal(t, bytecode, code)
}

func TestNewExecutable(t *testing.T) {
	t.Run("valid creation", func(t *testing.T) {
		content := "print('test')"
		bytecode := &starlarkLib.Program{}

		exe := newExecutable([]byte(content), bytecode)
		require.NotNil(t, exe)
		assert.Equal(t, content, exe.GetSource())
		assert.Equal(t, bytecode, exe.ByteCode)
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
}
