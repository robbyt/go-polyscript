package risor

import (
	"testing"

	risorCompiler "github.com/risor-io/risor/compiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewExecutableValid tests creating an Executable with valid content and bytecode
func TestNewExecutableValid(t *testing.T) {
	content := "print('Hello, World!')"
	bytecode := &risorCompiler.Code{}

	executable := newExecutable([]byte(content), bytecode)
	require.NotNil(t, executable)
	assert.Equal(t, content, executable.GetSource())
	assert.Equal(t, bytecode, executable.GetByteCode())
	assert.Equal(t, bytecode, executable.GetRisorByteCode())
}

// TestNewExecutableNilContent tests creating an Executable with nil content
func TestNewExecutableNilContent(t *testing.T) {
	bytecode := &risorCompiler.Code{}

	executable := newExecutable(nil, bytecode)
	require.Nil(t, executable)
}

// TestNewExecutableNilByteCode tests creating an Executable with nil bytecode
func TestNewExecutableNilByteCode(t *testing.T) {
	content := "print('Hello, World!')"

	executable := newExecutable([]byte(content), nil)
	require.Nil(t, executable)
}

// TestNewExecutableNilContentAndByteCode tests creating an Executable with nil content and bytecode
func TestNewExecutableNilContentAndByteCode(t *testing.T) {
	executable := newExecutable(nil, nil)
	require.Nil(t, executable)
}

// TestExecutable_GetBody tests the GetBody method of Executable
func TestExecutable_GetBody(t *testing.T) {
	content := "print('Hello, World!')"
	bytecode := &risorCompiler.Code{}
	executable := newExecutable([]byte(content), bytecode)
	require.NotNil(t, executable)

	body := executable.GetSource()
	assert.Equal(t, content, body)
}

// TestExecutable_GetByteCode tests the GetByteCode method of Executable
func TestExecutable_GetByteCode(t *testing.T) {
	content := "print('Hello, World!')"
	bytecode := &risorCompiler.Code{}
	executable := newExecutable([]byte(content), bytecode)
	require.NotNil(t, executable)

	code := executable.GetByteCode()
	assert.Equal(t, bytecode, code)

	// Test type assertion
	_, ok := code.(*risorCompiler.Code)
	assert.True(t, ok)
}

// TestExecutable_GetRisorByteCode tests the GetRisorByteCode method of Executable
func TestExecutable_GetRisorByteCode(t *testing.T) {
	content := "print('Hello, World!')"
	bytecode := &risorCompiler.Code{}
	executable := newExecutable([]byte(content), bytecode)
	require.NotNil(t, executable)

	code := executable.GetRisorByteCode()
	assert.Equal(t, bytecode, code)
}

func TestNewExecutable(t *testing.T) {
	t.Run("valid creation", func(t *testing.T) {
		content := "print('test')"
		bytecode := &risorCompiler.Code{}

		exe := newExecutable([]byte(content), bytecode)
		require.NotNil(t, exe)
		assert.Equal(t, content, exe.GetSource())
		assert.Equal(t, bytecode, exe.ByteCode)
	})

	t.Run("nil content", func(t *testing.T) {
		bytecode := &risorCompiler.Code{}
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
