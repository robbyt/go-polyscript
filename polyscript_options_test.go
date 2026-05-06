package polyscript_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/robbyt/go-polyscript/platform/script/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRisorOptionsAPI(t *testing.T) {
	t.Parallel()

	t.Run("FromString minimal", func(t *testing.T) {
		eval, err := polyscript.Risor(polyscript.FromString(`"hello"`))
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromString with static data", func(t *testing.T) {
		script := `{"name": ctx["name"], "length": len(ctx["name"])}`
		eval, err := polyscript.Risor(
			polyscript.FromString(script),
			polyscript.WithStaticData(map[string]any{"name": "World"}),
		)
		require.NoError(t, err)

		result, err := eval.Eval(t.Context())
		require.NoError(t, err)
		got, ok := result.Interface().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "World", got["name"])
	})

	t.Run("FromString with dynamic runtime data", func(t *testing.T) {
		script := `{"name": ctx["name"]}`
		eval, err := polyscript.Risor(polyscript.FromString(script))
		require.NoError(t, err)

		ctx, err := eval.AddDataToContext(t.Context(), map[string]any{"name": "Robert"})
		require.NoError(t, err)
		result, err := eval.Eval(ctx)
		require.NoError(t, err)
		got, ok := result.Interface().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Robert", got["name"])
	})

	t.Run("FromFile", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "script.risor")
		require.NoError(t, os.WriteFile(path, []byte(`{"ok": true}`), 0o600))

		eval, err := polyscript.Risor(polyscript.FromFile(path))
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("empty string surfaces error from source", func(t *testing.T) {
		_, err := polyscript.Risor(polyscript.FromString(""))
		require.Error(t, err)
	})

	t.Run("zero-value Source errors", func(t *testing.T) {
		_, err := polyscript.Risor(polyscript.Source{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "zero-value Source")
	})
}

func TestStarlarkOptionsAPI(t *testing.T) {
	t.Parallel()

	t.Run("FromString minimal", func(t *testing.T) {
		eval, err := polyscript.Starlark(polyscript.FromString(`_ = "hello"`))
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromString with static data", func(t *testing.T) {
		script := `result = {"name": ctx["name"]}
_ = result`
		eval, err := polyscript.Starlark(
			polyscript.FromString(script),
			polyscript.WithStaticData(map[string]any{"name": "World"}),
		)
		require.NoError(t, err)

		result, err := eval.Eval(t.Context())
		require.NoError(t, err)
		got, ok := result.Interface().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "World", got["name"])
	})
}

func TestExtismOptionsAPI(t *testing.T) {
	t.Parallel()

	t.Run("FromBytes with entry point", func(t *testing.T) {
		eval, err := polyscript.Extism(
			polyscript.FromBytes(wasmdata.TestModule),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
			polyscript.WithStaticData(map[string]any{"input": "World"}),
		)
		require.NoError(t, err)

		result, err := eval.Eval(t.Context())
		require.NoError(t, err)
		got, ok := result.Interface().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Hello, World!", got["greeting"])
	})

	t.Run("missing entry point errors", func(t *testing.T) {
		_, err := polyscript.Extism(polyscript.FromBytes(wasmdata.TestModule))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "entry point is required")
	})

	t.Run("missing entry point errors before resolving source", func(t *testing.T) {
		// The missing-entrypoint check must short-circuit before we try to
		// read empty bytes, so the user gets the more informative error.
		_, err := polyscript.Extism(polyscript.FromBytes(nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "entry point is required")
	})

	t.Run("FromFile", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.wasm")
		require.NoError(t, os.WriteFile(path, wasmdata.TestModule, 0o600))

		eval, err := polyscript.Extism(
			polyscript.FromFile(path),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
			polyscript.WithStaticData(map[string]any{"input": "Test"}),
		)
		require.NoError(t, err)

		result, err := eval.Eval(t.Context())
		require.NoError(t, err)
		got, ok := result.Interface().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Hello, Test!", got["greeting"])
	})
}

func TestFromLoader(t *testing.T) {
	t.Parallel()

	t.Run("wraps a custom loader", func(t *testing.T) {
		ldr, err := loader.NewFromString(`"ok"`)
		require.NoError(t, err)

		eval, err := polyscript.Risor(polyscript.FromLoader(ldr))
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("nil loader errors", func(t *testing.T) {
		_, err := polyscript.Risor(polyscript.FromLoader(nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil loader")
	})
}

func TestOptionsCompose(t *testing.T) {
	t.Parallel()

	t.Run("nil Option is ignored", func(t *testing.T) {
		eval, err := polyscript.Risor(
			polyscript.FromString(`"ok"`),
			nil,
			polyscript.WithStaticData(map[string]any{"k": "v"}),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("WithLogHandler accepts nil", func(t *testing.T) {
		eval, err := polyscript.Risor(
			polyscript.FromString(`"ok"`),
			polyscript.WithLogHandler(nil),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("later option wins for static data", func(t *testing.T) {
		eval, err := polyscript.Risor(
			polyscript.FromString(`{"k": ctx["k"]}`),
			polyscript.WithStaticData(map[string]any{"k": "first"}),
			polyscript.WithStaticData(map[string]any{"k": "second"}),
		)
		require.NoError(t, err)

		result, err := eval.Eval(t.Context())
		require.NoError(t, err)
		got, ok := result.Interface().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "second", got["k"])
	})
}

// Deprecated constructors still work. These tests should be removed when the
// deprecated functions are finally deleted in the v1 cleanup.
func TestDeprecatedConstructorsStillWork(t *testing.T) {
	t.Parallel()

	t.Run("FromRisorString", func(t *testing.T) {
		//nolint:staticcheck // Intentionally testing the deprecated function.
		eval, err := polyscript.FromRisorString(`"ok"`, nil)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromRisorStringWithData", func(t *testing.T) {
		//nolint:staticcheck // Intentionally testing the deprecated function.
		eval, err := polyscript.FromRisorStringWithData(`{"k": ctx["k"]}`,
			map[string]any{"k": "v"}, nil)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromExtismBytesWithData", func(t *testing.T) {
		//nolint:staticcheck // Intentionally testing the deprecated function.
		eval, err := polyscript.FromExtismBytesWithData(
			wasmdata.TestModule,
			map[string]any{"input": "World"},
			nil,
			wasmdata.EntrypointGreet,
		)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})
}

func TestSourceErrorPropagation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		source polyscript.Source
		want   string
	}{
		{"FromString empty", polyscript.FromString(""), "empty"},
		{"FromBytes empty", polyscript.FromBytes(nil), "empty"},
		{"FromFile relative", polyscript.FromFile("relative/path.risor"), "relative"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := polyscript.Risor(tc.source)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}
