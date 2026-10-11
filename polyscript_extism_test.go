package polyscript_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/extism/compiler"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var wasmData = wasmdata.TestModule

func TestExtismFromBytes(t *testing.T) {
	t.Parallel()

	t.Run("success with embedded wasmdata", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Extism](
			t.Context(),
			polyscript.FromBytes(wasmdata.TestModule),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("empty bytes", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Extism](
			t.Context(),
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
			t.Context(),
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
			t.Context(),
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
			t.Context(),
			polyscript.FromFile("non-existent-file.wasm"),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
		)
		require.Error(t, err)
	})

	t.Run("with static data", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Extism](
			t.Context(),
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
			t.Context(),
			polyscript.FromFile(wasmPath),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
			polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "Test User"}),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)

		result, err := eval.Eval(t.Context())
		require.NoError(t, err)
		require.NotNil(t, result)

		resultMap, err := result.AsMap()
		require.NoError(t, err, "Result should be a map")
		require.Contains(t, resultMap, "greeting")
		assert.Equal(t, "Hello, Test User!", resultMap["greeting"])

		evalNoInput, err := polyscript.New[polyscript.Extism](
			t.Context(),
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
		t.Context(),
		polyscript.FromFile(wasmPath),
		polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
		polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "Test User"}),
	)
	require.NoError(t, err)
	require.NotNil(t, eval)

	result, err := eval.Eval(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, err := result.AsMap()
	require.NoError(t, err, "Result should be a map")
	require.Contains(t, resultMap, "greeting")
	assert.Equal(t, "Hello, Test User!", resultMap["greeting"])
}

// TestExtismEvaluatorClose verifies Close releases the compiled plugin's
// wazero runtime: it is idempotent and later Evals fail with
// compiler.ErrExecutableClosed.
func TestExtismEvaluatorClose(t *testing.T) {
	t.Parallel()

	newEval := func(t *testing.T) platform.Evaluator {
		t.Helper()
		eval, err := polyscript.New[polyscript.Extism](
			t.Context(),
			polyscript.FromBytes(wasmdata.TestModule),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
			polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "Test User"}),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)
		return eval
	}

	t.Run("eval after close fails", func(t *testing.T) {
		eval := newEval(t)

		_, err := eval.Eval(t.Context())
		require.NoError(t, err, "Eval before Close should succeed")

		require.NoError(t, eval.Close(t.Context()))
		require.NoError(t, eval.Close(t.Context()), "second Close should be a no-op")

		result, err := eval.Eval(t.Context())
		require.ErrorIs(t, err, compiler.ErrExecutableClosed)
		assert.Nil(t, result)
	})

	t.Run("concurrent eval and close", func(t *testing.T) {
		eval := newEval(t)

		const workers = 8
		const evalsPerWorker = 10

		var wg sync.WaitGroup
		start := make(chan struct{})
		errs := make(chan error, workers*evalsPerWorker+2)
		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				for range evalsPerWorker {
					_, err := eval.Eval(t.Context())
					errs <- err
				}
			}()
		}

		// Two concurrent Closes race the Evals and each other.
		for range 2 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				errs <- eval.Close(t.Context())
			}()
		}

		close(start)
		wg.Wait()
		close(errs)

		for err := range errs {
			if err != nil {
				require.ErrorIs(t, err, compiler.ErrExecutableClosed)
			}
		}

		_, err := eval.Eval(t.Context())
		require.ErrorIs(t, err, compiler.ErrExecutableClosed)
	})
}
