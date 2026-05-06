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

func TestNewRisor(t *testing.T) {
	t.Parallel()

	t.Run("FromString minimal", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Risor](polyscript.FromString(`"hello"`))
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromString with static data", func(t *testing.T) {
		script := `{"name": ctx["name"], "length": len(ctx["name"])}`
		eval, err := polyscript.New[polyscript.Risor](
			polyscript.FromString(script),
			polyscript.WithStaticData[polyscript.Risor](map[string]any{"name": "World"}),
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
		eval, err := polyscript.New[polyscript.Risor](polyscript.FromString(script))
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

		eval, err := polyscript.New[polyscript.Risor](polyscript.FromFile(path))
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("empty string surfaces error from source", func(t *testing.T) {
		_, err := polyscript.New[polyscript.Risor](polyscript.FromString(""))
		require.Error(t, err)
	})

	t.Run("zero-value Source errors", func(t *testing.T) {
		_, err := polyscript.New[polyscript.Risor](polyscript.Source{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "zero-value Source")
	})
}

func TestNewStarlark(t *testing.T) {
	t.Parallel()

	t.Run("FromString minimal", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Starlark](polyscript.FromString(`_ = "hello"`))
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromString with static data", func(t *testing.T) {
		script := `result = {"name": ctx["name"]}
_ = result`
		eval, err := polyscript.New[polyscript.Starlark](
			polyscript.FromString(script),
			polyscript.WithStaticData[polyscript.Starlark](map[string]any{"name": "World"}),
		)
		require.NoError(t, err)

		result, err := eval.Eval(t.Context())
		require.NoError(t, err)
		got, ok := result.Interface().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "World", got["name"])
	})
}

func TestNewExtism(t *testing.T) {
	t.Parallel()

	t.Run("FromBytes with entry point", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Extism](
			polyscript.FromBytes(wasmdata.TestModule),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
			polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "World"}),
		)
		require.NoError(t, err)

		result, err := eval.Eval(t.Context())
		require.NoError(t, err)
		got, ok := result.Interface().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Hello, World!", got["greeting"])
	})

	t.Run("missing entry point errors", func(t *testing.T) {
		_, err := polyscript.New[polyscript.Extism](polyscript.FromBytes(wasmdata.TestModule))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "entry point is required")
	})

	t.Run("missing entry point errors before resolving source", func(t *testing.T) {
		// The missing-entrypoint check must short-circuit before we try to
		// read empty bytes, so the user gets the more informative error.
		_, err := polyscript.New[polyscript.Extism](polyscript.FromBytes(nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "entry point is required")
	})

	t.Run("FromFile", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.wasm")
		require.NoError(t, os.WriteFile(path, wasmdata.TestModule, 0o600))

		eval, err := polyscript.New[polyscript.Extism](
			polyscript.FromFile(path),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
			polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "Test"}),
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

		eval, err := polyscript.New[polyscript.Risor](polyscript.FromLoader(ldr))
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("nil loader errors", func(t *testing.T) {
		_, err := polyscript.New[polyscript.Risor](polyscript.FromLoader(nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil loader")
	})

	t.Run("typed-nil loader errors instead of panicking", func(t *testing.T) {
		// Demonstrates the typed-nil interface gotcha: l is an interface
		// holding a non-nil type descriptor (*loader.FromString) but a nil
		// concrete value. A naive nil-check would let this through and the
		// engine constructor would panic on first method call.
		var typedNil *loader.FromString
		var l loader.Loader = typedNil

		_, err := polyscript.New[polyscript.Risor](polyscript.FromLoader(l))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil loader")
	})
}

func TestOptionsCompose(t *testing.T) {
	t.Parallel()

	t.Run("nil Option is ignored", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Risor](
			polyscript.FromString(`"ok"`),
			nil,
			polyscript.WithStaticData[polyscript.Risor](map[string]any{"k": "v"}),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("WithLogHandler accepts nil", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Risor](
			polyscript.FromString(`"ok"`),
			polyscript.WithLogHandler[polyscript.Risor](nil),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("later option wins for static data", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Risor](
			polyscript.FromString(`{"k": ctx["k"]}`),
			polyscript.WithStaticData[polyscript.Risor](map[string]any{"k": "first"}),
			polyscript.WithStaticData[polyscript.Risor](map[string]any{"k": "second"}),
		)
		require.NoError(t, err)

		result, err := eval.Eval(t.Context())
		require.NoError(t, err)
		got, ok := result.Interface().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "second", got["k"])
	})
}

// FromBytes must isolate compilation from caller-side mutations to the input
// slice. This is the regression test for the aliasing footgun called out in
// the Copilot review.
func TestFromBytesIsolatesCallerSlice(t *testing.T) {
	t.Parallel()

	original := make([]byte, len(wasmdata.TestModule))
	copy(original, wasmdata.TestModule)

	src := polyscript.FromBytes(original)

	// Scribble over the caller's slice before the engine resolves the source.
	for i := range original {
		original[i] = 0xff
	}

	eval, err := polyscript.New[polyscript.Extism](
		src,
		polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
		polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "World"}),
	)
	require.NoError(t, err, "FromBytes should snapshot the caller's slice")
	require.NotNil(t, eval)

	result, err := eval.Eval(t.Context())
	require.NoError(t, err)
	got, ok := result.Interface().(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Hello, World!", got["greeting"])
}

// Deprecated constructors still work. This pins the deprecation contract for
// the duration of the deprecation window; remove when those functions are
// finally deleted in the v1 cleanup.
func TestDeprecatedConstructorsStillWork(t *testing.T) {
	t.Parallel()

	t.Run("FromRisorString", func(t *testing.T) {
		eval, err := polyscript.FromRisorString(`"ok"`, nil)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("FromRisorStringWithData", func(t *testing.T) {
		eval, err := polyscript.FromRisorStringWithData(
			`{"k": ctx["k"]}`, map[string]any{"k": "v"}, nil)
		require.NoError(t, err)
		require.NotNil(t, eval)
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
			_, err := polyscript.New[polyscript.Risor](tc.source)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}
