package polyscript_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var wasmData = wasmdata.TestModule

func TestExtismFromBytes(t *testing.T) {
	t.Parallel()

	t.Run("success with embedded wasmdata", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Extism](
			polyscript.FromBytes(wasmdata.TestModule),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("empty bytes", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Extism](
			polyscript.FromBytes([]byte{}),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
		)
		require.Error(t, err)
		require.Nil(t, eval)
	})
}

func TestExtismFromBytesWithData(t *testing.T) {
	t.Parallel()

	t.Run("success with static data", func(t *testing.T) {
		staticData := map[string]any{
			"version": "1.0.0",
			"config": map[string]any{
				"timeout": 30,
				"retry":   true,
			},
		}

		eval, err := polyscript.New[polyscript.Extism](
			polyscript.FromBytes(wasmdata.TestModule),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
			polyscript.WithStaticData[polyscript.Extism](staticData),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)

		runtimeData := map[string]any{"input": "test user"}
		ctx, err := eval.AddDataToContext(t.Context(), runtimeData)
		require.NoError(t, err)

		response, err := eval.Eval(ctx)
		require.NoError(t, err)
		require.NotNil(t, response)
	})
}

func TestExtismFromFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "main.wasm")
	require.NoError(t, os.WriteFile(wasmPath, wasmData, 0o644))

	t.Run("valid", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Extism](
			polyscript.FromFile(wasmPath),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)

		ctx := context.WithValue(t.Context(), constants.EvalData, map[string]any{
			"input": "Test User",
		})

		result, err := eval.Eval(ctx)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("invalid path", func(t *testing.T) {
		_, err := polyscript.New[polyscript.Extism](
			polyscript.FromFile("non-existent-file.wasm"),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
		)
		require.Error(t, err)
	})

	t.Run("with static data", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Extism](
			polyscript.FromFile(wasmPath),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
			polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "Test User"}),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})
}

func TestExtismDataIntegrationScenarios(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "main.wasm")
	require.NoError(t, os.WriteFile(wasmPath, wasmData, 0o644))

	t.Run("ExtismWithData", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Extism](
			polyscript.FromFile(wasmPath),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
			polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "Test User"}),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)

		result, err := eval.Eval(t.Context())
		require.NoError(t, err)
		require.NotNil(t, result)

		resultMap, ok := result.Interface().(map[string]any)
		require.True(t, ok, "Result should be a map")
		require.Contains(t, resultMap, "greeting")
		assert.Equal(t, "Hello, Test User!", resultMap["greeting"])

		evalNoInput, err := polyscript.New[polyscript.Extism](
			polyscript.FromFile(wasmPath),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
			polyscript.WithStaticData[polyscript.Extism](map[string]any{
				"app_version": "1.0.0",
				"config":      map[string]any{"timeout": 30},
			}),
		)
		require.NoError(t, err)
		require.NotNil(t, evalNoInput)

		_, err = evalNoInput.Eval(t.Context())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "input string is empty")
	})
}

func TestExtismEvalEndToEnd(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "main.wasm")
	require.NoError(t, os.WriteFile(wasmPath, wasmData, 0o644))

	eval, err := polyscript.New[polyscript.Extism](
		polyscript.FromFile(wasmPath),
		polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
		polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "Test User"}),
	)
	require.NoError(t, err)
	require.NotNil(t, eval)

	result, err := eval.Eval(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, ok := result.Interface().(map[string]any)
	require.True(t, ok, "Result should be a map")
	require.Contains(t, resultMap, "greeting")
	assert.Equal(t, "Hello, Test User!", resultMap["greeting"])
}
