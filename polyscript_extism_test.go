package polyscript_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

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

// TestNew_WithLifetime checks that polyscript.WithLifetime reaches each
// engine: an evaluator that holds resources closes when its lifetime ends,
// one that holds none keeps evaluating, and an already-done lifetime fails
// construction on every engine.
func TestNew_WithLifetime(t *testing.T) {
	t.Parallel()

	newExtism := func(setup, lifetime context.Context) (platform.Evaluator, error) {
		return polyscript.New[polyscript.Extism](
			setup,
			polyscript.FromBytes(wasmdata.TestModule),
			polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
			polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "World"}),
			polyscript.WithLifetime[polyscript.Extism](lifetime),
		)
	}
	newRisor := func(setup, lifetime context.Context) (platform.Evaluator, error) {
		return polyscript.New[polyscript.Risor](
			setup, polyscript.FromString(`"ok"`), polyscript.WithLifetime[polyscript.Risor](lifetime),
		)
	}
	newStarlark := func(setup, lifetime context.Context) (platform.Evaluator, error) {
		return polyscript.New[polyscript.Starlark](
			setup, polyscript.FromString(`_ = "ok"`), polyscript.WithLifetime[polyscript.Starlark](lifetime),
		)
	}

	tests := []struct {
		name        string
		create      func(setup, lifetime context.Context) (platform.Evaluator, error)
		closesOnEnd bool
	}{
		{"Extism", newExtism, true},
		{"Risor", newRisor, false},
		{"Starlark", newStarlark, false},
	}

	for _, tc := range tests {
		t.Run(tc.name+" lifetime end", func(t *testing.T) {
			setup, cancelSetup := context.WithCancel(t.Context())
			lifetime, cancelLifetime := context.WithCancel(t.Context())
			defer cancelLifetime()

			eval, err := tc.create(setup, lifetime)
			require.NoError(t, err)
			cancelSetup()
			// A hook wrongly bound to the setup ctx runs on its own goroutine,
			// so give it time to close the evaluator before checking.
			require.Never(t, func() bool {
				_, err := eval.Eval(t.Context())
				return err != nil
			}, 50*time.Millisecond, 5*time.Millisecond, "the setup ctx must not bound the evaluator's lifetime")
			_, err = eval.Eval(t.Context())
			require.NoError(t, err, "the setup ctx must not bound the evaluator's lifetime")

			cancelLifetime()
			if !tc.closesOnEnd {
				_, err = eval.Eval(t.Context())
				require.NoError(t, err, "an engine without resources keeps evaluating")
				return
			}
			require.Eventually(t, func() bool {
				_, err := eval.Eval(t.Context())
				return errors.Is(err, compiler.ErrExecutableClosed)
			}, 5*time.Second, time.Millisecond, "the evaluator should close when its lifetime ends")
		})

		t.Run(tc.name+" already-done lifetime", func(t *testing.T) {
			lifetime, cancelLifetime := context.WithCancel(t.Context())
			cancelLifetime()

			eval, err := tc.create(t.Context(), lifetime)
			require.ErrorIs(t, err, context.Canceled)
			if eval != nil {
				t.Errorf("failed New should return an untyped nil evaluator, got %#v", eval)
			}
		})
	}
}
