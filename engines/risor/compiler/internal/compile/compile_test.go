package compile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCompileSuccess tests the successful compilation of valid script content
func TestCompileSuccess(t *testing.T) {
	scriptContent := `true`

	code, err := Compile(&scriptContent)
	require.NoError(t, err)
	require.NotNil(t, code)
}

// TestCompileSyntaxError tests the compilation failure due to syntax errors
func TestCompileSyntaxError(t *testing.T) {
	scriptContent := `
		print("Hello, World!
	`

	code, err := Compile(&scriptContent)
	require.Error(t, err)
	require.Nil(t, code)
	require.ErrorIs(t, err, ErrCompileFailed)
}

// TestCompileWithGlobals tests the compilation with custom global names
func TestCompileWithGlobals(t *testing.T) {
	scriptContent := `
		print(request)
	`

	globals := []string{"request"}
	code, err := CompileWithGlobals(&scriptContent, globals)
	require.NoError(t, err)
	require.NotNil(t, code)
}

// TestCompileNilContent tests the handling of nil script content
func TestCompileNilContent(t *testing.T) {
	code, err := Compile(nil)
	require.Error(t, err)
	require.Nil(t, code)
	require.ErrorIs(t, err, ErrContentNil)
}

// TestCompileWithGlobalsNilContent tests the handling of nil script content with globals
func TestCompileWithGlobalsNilContent(t *testing.T) {
	globals := []string{"request"}
	code, err := CompileWithGlobals(nil, globals)
	require.Error(t, err)
	require.Nil(t, code)
	require.ErrorIs(t, err, ErrContentNil)
}

// TestCompileWithGlobalsSyntaxError tests the compilation failure due to syntax errors with globals
func TestCompileWithGlobalsSyntaxError(t *testing.T) {
	scriptContent := `
		print(request
	`

	globals := []string{"request"}
	code, err := CompileWithGlobals(&scriptContent, globals)
	require.Error(t, err)
	require.Nil(t, code)
	require.ErrorIs(t, err, ErrCompileFailed)
}
