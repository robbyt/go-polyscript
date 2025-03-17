package risor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCompileSuccess tests the successful compilation of valid script content
func TestCompileSuccess(t *testing.T) {
	scriptContent := `true`

	code, err := compile(&scriptContent)
	require.NoError(t, err)
	require.NotNil(t, code)
}

// TestCompileSyntaxError tests the compilation failure due to syntax errors
func TestCompileSyntaxError(t *testing.T) {
	scriptContent := `
		print("Hello, World!
	`

	code, err := compile(&scriptContent)
	require.Error(t, err)
	require.Nil(t, code)
	require.Contains(t, err.Error(), "compilation:")
}

// TestCompileWithGlobals tests the compilation with custom global names
func TestCompileWithGlobals(t *testing.T) {
	scriptContent := `
		print(request)
	`

	globals := []string{"request"}
	code, err := compileWithGlobals(&scriptContent, globals)
	require.NoError(t, err)
	require.NotNil(t, code)
}

// TestCompileNilContent tests the handling of nil script content
func TestCompileNilContent(t *testing.T) {
	code, err := compile(nil)
	require.Error(t, err)
	require.Nil(t, code)
	require.Contains(t, err.Error(), "script content is nil")
}

// TestCompileWithGlobalsNilContent tests the handling of nil script content with globals
func TestCompileWithGlobalsNilContent(t *testing.T) {
	globals := []string{"request"}
	code, err := compileWithGlobals(nil, globals)
	require.Error(t, err)
	require.Nil(t, code)
	require.Contains(t, err.Error(), "script content is nil")
}

// TestCompileWithGlobalsSyntaxError tests the compilation failure due to syntax errors with globals
func TestCompileWithGlobalsSyntaxError(t *testing.T) {
	scriptContent := `
		print(request
	`

	globals := []string{"request"}
	code, err := compileWithGlobals(&scriptContent, globals)
	require.Error(t, err)
	require.Nil(t, code)
	require.Contains(t, err.Error(), "compilation:")
}
