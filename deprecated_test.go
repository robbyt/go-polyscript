package polyscript_test

// Smoke tests for the deprecated FromXxx* constructors in deprecated.go.
//
// Each test calls the function with a minimal valid input and asserts no
// error. The bodies are trivial wrappers that delegate to the per-engine
// packages, so we don't re-validate evaluation behavior here — the goal is
// to pin the deprecation contract for the deprecation window. Delete this
// file alongside deprecated.go when the deprecated functions are removed
// in v1.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/stretchr/testify/require"
)

func writeTempFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, content, 0o600))
	return path
}

func TestDeprecatedRisorConstructors(t *testing.T) {
	t.Parallel()

	t.Run("FromRisorString", func(t *testing.T) {
		eval, err := polyscript.FromRisorString(`"ok"`, nil)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromRisorString empty errors", func(t *testing.T) {
		_, err := polyscript.FromRisorString("", nil)
		require.Error(t, err)
	})

	t.Run("FromRisorStringWithData", func(t *testing.T) {
		eval, err := polyscript.FromRisorStringWithData(
			`{"k": ctx["k"]}`, map[string]any{"k": "v"}, nil)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromRisorStringWithData empty errors", func(t *testing.T) {
		_, err := polyscript.FromRisorStringWithData("", map[string]any{}, nil)
		require.Error(t, err)
	})

	t.Run("FromRisorFile", func(t *testing.T) {
		path := writeTempFile(t, "ok.risor", []byte(`"ok"`))
		eval, err := polyscript.FromRisorFile(path, nil)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromRisorFile missing errors", func(t *testing.T) {
		_, err := polyscript.FromRisorFile("/this/does/not/exist.risor", nil)
		require.Error(t, err)
	})

	t.Run("FromRisorFileWithData", func(t *testing.T) {
		path := writeTempFile(t, "ok.risor", []byte(`{"k": ctx["k"]}`))
		eval, err := polyscript.FromRisorFileWithData(path, map[string]any{"k": "v"}, nil)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromRisorFileWithData missing errors", func(t *testing.T) {
		_, err := polyscript.FromRisorFileWithData("/no/such.risor", map[string]any{}, nil)
		require.Error(t, err)
	})
}

func TestDeprecatedStarlarkConstructors(t *testing.T) {
	t.Parallel()

	t.Run("FromStarlarkString", func(t *testing.T) {
		eval, err := polyscript.FromStarlarkString(`_ = "ok"`, nil)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromStarlarkString empty errors", func(t *testing.T) {
		_, err := polyscript.FromStarlarkString("", nil)
		require.Error(t, err)
	})

	t.Run("FromStarlarkStringWithData", func(t *testing.T) {
		script := "_ = ctx\n"
		eval, err := polyscript.FromStarlarkStringWithData(
			script, map[string]any{"k": "v"}, nil)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromStarlarkStringWithData empty errors", func(t *testing.T) {
		_, err := polyscript.FromStarlarkStringWithData("", map[string]any{}, nil)
		require.Error(t, err)
	})

	t.Run("FromStarlarkFile", func(t *testing.T) {
		path := writeTempFile(t, "ok.star", []byte(`_ = "ok"`))
		eval, err := polyscript.FromStarlarkFile(path, nil)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromStarlarkFile missing errors", func(t *testing.T) {
		_, err := polyscript.FromStarlarkFile("/no/such.star", nil)
		require.Error(t, err)
	})

	t.Run("FromStarlarkFileWithData", func(t *testing.T) {
		path := writeTempFile(t, "ok.star", []byte("_ = ctx\n"))
		eval, err := polyscript.FromStarlarkFileWithData(
			path, map[string]any{"k": "v"}, nil)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromStarlarkFileWithData missing errors", func(t *testing.T) {
		_, err := polyscript.FromStarlarkFileWithData("/no/such.star", map[string]any{}, nil)
		require.Error(t, err)
	})
}

func TestDeprecatedExtismConstructors(t *testing.T) {
	t.Parallel()

	t.Run("FromExtismBytes", func(t *testing.T) {
		eval, err := polyscript.FromExtismBytes(
			wasmdata.TestModule, nil, wasmdata.EntrypointGreet)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromExtismBytes empty errors", func(t *testing.T) {
		_, err := polyscript.FromExtismBytes(nil, nil, wasmdata.EntrypointGreet)
		require.Error(t, err)
	})

	t.Run("FromExtismBytesWithData", func(t *testing.T) {
		eval, err := polyscript.FromExtismBytesWithData(
			wasmdata.TestModule,
			map[string]any{"input": "World"},
			nil,
			wasmdata.EntrypointGreet,
		)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromExtismBytesWithData empty errors", func(t *testing.T) {
		_, err := polyscript.FromExtismBytesWithData(
			nil, map[string]any{}, nil, wasmdata.EntrypointGreet)
		require.Error(t, err)
	})

	t.Run("FromExtismFile", func(t *testing.T) {
		path := writeTempFile(t, "main.wasm", wasmdata.TestModule)
		eval, err := polyscript.FromExtismFile(path, nil, wasmdata.EntrypointGreet)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromExtismFile missing errors", func(t *testing.T) {
		_, err := polyscript.FromExtismFile("/no/such.wasm", nil, wasmdata.EntrypointGreet)
		require.Error(t, err)
	})

	t.Run("FromExtismFileWithData", func(t *testing.T) {
		path := writeTempFile(t, "main.wasm", wasmdata.TestModule)
		eval, err := polyscript.FromExtismFileWithData(
			path,
			map[string]any{"input": "Test User"},
			nil,
			wasmdata.EntrypointGreet,
		)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromExtismFileWithData missing errors", func(t *testing.T) {
		_, err := polyscript.FromExtismFileWithData(
			"/no/such.wasm", map[string]any{}, nil, wasmdata.EntrypointGreet)
		require.Error(t, err)
	})
}
