package polyscript_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockPreparer implements data.Setter for testing
type mockPreparer struct {
	mock.Mock
}

func (m *mockPreparer) AddDataToContext(
	ctx context.Context,
	data map[string]any,
) (context.Context, error) {
	args := m.Called(ctx, data)
	return args.Get(0).(context.Context), args.Error(1)
}

// evalAndExtractMap runs evaluation and extracts result as a map[string]any.
func evalAndExtractMap(
	t *testing.T,
	ctx context.Context,
	evaluator platform.EvalOnly,
) (map[string]any, error) {
	t.Helper()

	result, err := evaluator.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("script evaluation failed: %w", err)
	}

	val := result.Interface()
	if val == nil {
		return map[string]any{}, nil
	}

	data, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("result is not a map: %T", val)
	}

	return data, nil
}

// prepareAndEval combines context preparation and evaluation in a single operation.
func prepareAndEval(
	t *testing.T,
	ctx context.Context,
	evaluator platform.Evaluator,
	runtimeData map[string]any,
) (platform.EvaluatorResponse, error) {
	t.Helper()

	enrichedCtx, err := evaluator.AddDataToContext(ctx, runtimeData)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare context: %w", err)
	}

	result, err := evaluator.Eval(enrichedCtx)
	if err != nil {
		return nil, fmt.Errorf("script evaluation failed: %w", err)
	}

	return result, nil
}

// stringEngineFn is a small adapter so a table-driven test can mix engines that
// share the (script-content, log-handler) shape.
type stringEngineFn func(ctx context.Context, content string, h slog.Handler) (platform.Evaluator, error)

func risorString(ctx context.Context, content string, h slog.Handler) (platform.Evaluator, error) {
	return polyscript.New[polyscript.Risor](ctx, polyscript.FromString(content), polyscript.WithLogHandler[polyscript.Risor](h))
}

func starlarkString(ctx context.Context, content string, h slog.Handler) (platform.Evaluator, error) {
	return polyscript.New[polyscript.Starlark](ctx, polyscript.FromString(content), polyscript.WithLogHandler[polyscript.Starlark](h))
}

func TestEngineEvaluatorsFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		creator stringEngineFn
	}{
		{"Starlark", `print("Hello, World!")`, starlarkString},
		{"Risor", `"Hello, World!"`, risorString},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			eval, err := tc.creator(t.Context(), tc.content, nil)
			require.NoError(t, err)
			require.NotNil(t, eval)
		})
	}
}

func TestFromStringValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string
		creator     stringEngineFn
		expectError bool
	}{
		{"Starlark valid", `print("Hello, World!")`, starlarkString, false},
		{"Risor valid", `"Hello, World!"`, risorString, false},
		{"Starlark empty", "", starlarkString, true},
		{"Risor empty", "", risorString, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			eval, err := tc.creator(t.Context(), tc.content, nil)
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, eval)
		})
	}
}

func TestFromFileLoaders(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	risorPath := filepath.Join(tmpDir, "test.risor")
	starlarkPath := filepath.Join(tmpDir, "test.star")

	risorContent := `{ "message": "Hello from Risor!" }`
	require.NoError(t, os.WriteFile(risorPath, []byte(risorContent), 0o644))

	starlarkContent := `result = {"message": "Hello from Starlark!"}
_ = result`
	require.NoError(t, os.WriteFile(starlarkPath, []byte(starlarkContent), 0o644))

	t.Run("Risor file - valid", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Risor](t.Context(), polyscript.FromFile(risorPath))
		require.NoError(t, err)
		require.NotNil(t, eval)

		result, err := eval.Eval(t.Context())
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("Risor file - invalid path", func(t *testing.T) {
		_, err := polyscript.New[polyscript.Risor](t.Context(), polyscript.FromFile("non-existent-file.risor"))
		require.Error(t, err)
	})

	t.Run("Risor file - with static data", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Risor](
			t.Context(),
			polyscript.FromFile(risorPath),
			polyscript.WithStaticData[polyscript.Risor](map[string]any{"test_key": "test_value"}),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})

	t.Run("Starlark file - valid", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Starlark](t.Context(), polyscript.FromFile(starlarkPath))
		require.NoError(t, err)
		require.NotNil(t, eval)

		result, err := eval.Eval(t.Context())
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("Starlark file - invalid path", func(t *testing.T) {
		_, err := polyscript.New[polyscript.Starlark](t.Context(), polyscript.FromFile("non-existent-file.star"))
		require.Error(t, err)
	})

	t.Run("Starlark file - with static data", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Starlark](
			t.Context(),
			polyscript.FromFile(starlarkPath),
			polyscript.WithStaticData[polyscript.Starlark](map[string]any{"test_key": "test_value"}),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)
	})
}

func TestDataProviders(t *testing.T) {
	t.Parallel()

	t.Run("withCompositeProvider", func(t *testing.T) {
		script := `print(ctx["static_key"], ", ", ctx["dynamic_key"])`

		eval, err := polyscript.New[polyscript.Starlark](
			t.Context(),
			polyscript.FromString(script),
			polyscript.WithStaticData[polyscript.Starlark](map[string]any{"static_key": "static_value"}),
		)
		require.NoError(t, err)
		require.NotNil(t, eval)

		ctx := t.Context()
		dynamicData := map[string]any{"dynamic_key": "dynamic_value"}
		enrichedCtx, err := eval.AddDataToContext(ctx, dynamicData)
		require.NoError(t, err)

		_, err = eval.Eval(enrichedCtx)
		require.NoError(t, err)
	})
}

func TestEvalHelpers(t *testing.T) {
	t.Parallel()

	t.Run("PrepareAndEval", func(t *testing.T) {
		script := `
            let name = ctx["name"]
            {
                "message": "Hello, " + name + "!",
                "length": len(name)
            }
        `

		eval, err := polyscript.New[polyscript.Risor](
			t.Context(),
			polyscript.FromString(script),
			polyscript.WithStaticData[polyscript.Risor](map[string]any{}),
		)
		require.NoError(t, err)

		result, err := prepareAndEval(t, t.Context(), eval, map[string]any{"name": "World"})
		require.NoError(t, err)
		require.NotNil(t, result)

		resultMap, err := result.AsMap()
		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", resultMap["message"])

		length := resultMap["length"]
		require.NotNil(t, length, "length field should be present")
		switch v := length.(type) {
		case int64:
			assert.Equal(t, int64(5), v, "length should be 5")
		case float64:
			assert.InDelta(t, float64(5), v, 0.0001, "length should be 5")
		default:
			t.Errorf("length is unexpected type %T", v)
		}

		t.Run("AddDataToContext error", func(t *testing.T) {
			mockPrepCtx := &mockPreparer{}
			mockEval := &mockEvaluator{}

			ctx := t.Context()
			d := map[string]any{"name": "World"}

			mockPrepCtx.On("AddDataToContext", ctx, mock.Anything).
				Return(ctx, errors.New("prepare error"))

			mockEvalWithPrep := struct {
				platform.EvalOnly
				data.Setter
			}{
				EvalOnly: mockEval,
				Setter:   mockPrepCtx,
			}

			_, err := prepareAndEval(t, ctx, mockEvalWithPrep, d)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to prepare context")
			mockPrepCtx.AssertExpectations(t)
		})

		t.Run("Eval error", func(t *testing.T) {
			mockPrepCtx := &mockPreparer{}
			mockEval := &mockEvaluator{}

			ctx := t.Context()
			d := map[string]any{"name": "World"}

			type contextKey string
			testKey := contextKey("test-key")
			enrichedCtx := context.WithValue(ctx, testKey, "test-value")
			mockPrepCtx.On("AddDataToContext", ctx, mock.Anything).Return(enrichedCtx, nil)

			mockEval.On("Eval", enrichedCtx).
				Return((*mockEvaluatorResponse)(nil), errors.New("eval error"))

			mockEvalWithPrep := struct {
				platform.EvalOnly
				data.Setter
			}{
				EvalOnly: mockEval,
				Setter:   mockPrepCtx,
			}

			_, err := prepareAndEval(t, ctx, mockEvalWithPrep, d)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "script evaluation failed")
			mockPrepCtx.AssertExpectations(t)
			mockEval.AssertExpectations(t)
		})
	})

	t.Run("EvalAndExtractMap", func(t *testing.T) {
		script := `
            {
                "message": "Hello, Static!",
                "length": 12
            }
        `

		eval, err := polyscript.New[polyscript.Risor](
			t.Context(),
			polyscript.FromString(script),
			polyscript.WithStaticData[polyscript.Risor](map[string]any{}),
		)
		require.NoError(t, err)

		resultMap, err := evalAndExtractMap(t, t.Context(), eval)
		require.NoError(t, err)

		assert.Equal(t, "Hello, Static!", resultMap["message"])

		length := resultMap["length"]
		require.NotNil(t, length, "length field should be present")
		switch v := length.(type) {
		case int64:
			assert.Equal(t, int64(12), v, "length should be 12")
		case float64:
			assert.InDelta(t, float64(12), v, 0.0001, "length should be 12")
		default:
			t.Errorf("length is unexpected type %T", v)
		}

		nilEval, err := polyscript.New[polyscript.Risor](t.Context(), polyscript.FromString(`nil`))
		require.NoError(t, err)
		nilResult, err := evalAndExtractMap(t, t.Context(), nilEval)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{}, nilResult)

		numEval, err := polyscript.New[polyscript.Risor](t.Context(), polyscript.FromString(`42`))
		require.NoError(t, err)
		_, err = evalAndExtractMap(t, t.Context(), numEval)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "result is not a map")

		t.Run("Eval error", func(t *testing.T) {
			mockEval := &mockEvaluator{}
			ctx := t.Context()

			mockEval.On("Eval", ctx).
				Return((*mockEvaluatorResponse)(nil), errors.New("eval error"))

			_, err := evalAndExtractMap(t, ctx, mockEval)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "script evaluation failed")
			mockEval.AssertExpectations(t)
		})
	})
}

func TestDataIntegrationScenarios(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	starlarkPath := filepath.Join(tmpDir, "test.star")

	starlarkFileContent := `# Simple Starlark script
result = {
    "message": "Hello, " + ctx["name"] + " (v" + ctx["app_version"] + ")",
    "timeout": ctx["config"]["timeout"]
}
_ = result`
	require.NoError(t, os.WriteFile(starlarkPath, []byte(starlarkFileContent), 0o644))

	staticData := map[string]any{
		"app_version": "1.0.0",
		"config": map[string]any{
			"timeout": 30,
		},
	}

	t.Run("RisorWithData", func(t *testing.T) {
		risorScript := `
            let version = ctx["app_version"]
            let timeout = ctx["config"]["timeout"]
            let name = ctx["name"]

            {
                "message": "Hello, " + name + " (v" + version + ")",
                "timeout": timeout
            }
        `

		eval, err := polyscript.New[polyscript.Risor](
			t.Context(),
			polyscript.FromString(risorScript),
			polyscript.WithStaticData[polyscript.Risor](staticData),
		)
		require.NoError(t, err)

		ctx := t.Context()
		dynamicData := map[string]any{"name": "Risor User"}
		enrichedCtx, err := eval.AddDataToContext(ctx, dynamicData)
		require.NoError(t, err)

		result, err := eval.Eval(enrichedCtx)
		require.NoError(t, err)

		resultMap, err := result.AsMap()
		require.NoError(t, err)
		assert.Equal(t, "Hello, Risor User (v1.0.0)", resultMap["message"])

		timeout := resultMap["timeout"]
		require.NotNil(t, timeout, "timeout field should be present")
		switch v := timeout.(type) {
		case int64:
			assert.Equal(t, int64(30), v, "timeout should be 30")
		case float64:
			assert.InDelta(t, float64(30), v, 0.0001, "timeout should be 30")
		default:
			t.Errorf("timeout is unexpected type %T", v)
		}
	})

	t.Run("StarlarkWithData", func(t *testing.T) {
		eval, err := polyscript.New[polyscript.Starlark](
			t.Context(),
			polyscript.FromFile(starlarkPath),
			polyscript.WithStaticData[polyscript.Starlark](staticData),
		)
		require.NoError(t, err)

		ctx := t.Context()
		dynamicData := map[string]any{"name": "Starlark User"}
		enrichedCtx, err := eval.AddDataToContext(ctx, dynamicData)
		require.NoError(t, err)

		result, err := eval.Eval(enrichedCtx)
		require.NoError(t, err)

		resultMap, err := result.AsMap()
		require.NoError(t, err)
		assert.Equal(t, "Hello, Starlark User (v1.0.0)", resultMap["message"])

		starlarkTimeout := resultMap["timeout"]
		require.NotNil(t, starlarkTimeout, "timeout field should be present")
		assert.Equal(t, int64(30), starlarkTimeout, "timeout should be 30")
	})
}

func TestCreateEvaluatorEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("Empty Script Content Error", func(t *testing.T) {
		_, err := polyscript.New[polyscript.Risor](t.Context(), polyscript.FromString(""))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("Invalid Path Error", func(t *testing.T) {
		_, err := polyscript.New[polyscript.Risor](t.Context(), polyscript.FromFile("/path/does/not/exist.risor"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	})

	t.Run("InvalidScriptTest", func(t *testing.T) {
		_, err := polyscript.New[polyscript.Risor](t.Context(), polyscript.FromString("this is not valid risor code }{"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "compile")
	})
}

// MockStringLoader is a simple loader.Loader implementation used for the
// hand-rolled "FromLoader path with a non-Loader source" smoke test.
type MockStringLoader struct{}

func (m *MockStringLoader) GetReader() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("return 0")), nil
}

func (m *MockStringLoader) GetSourceURL() *url.URL {
	return nil
}

func TestFromStringLoader(t *testing.T) {
	t.Parallel()

	t.Run("ExtismStringNotSupported", func(t *testing.T) {
		// Verify a string loader can be created — Extism would reject the
		// content as non-WASM if we tried to use it, but here we just
		// confirm the loader plumbing works.
		content := "test"
		l, err := loader.NewFromString(content)
		require.NoError(t, err)
		require.NotNil(t, l)
		require.NotNil(t, l.GetSourceURL())
	})
}

// TestNew_CompileCancelled proves the headline behavior of #145: a
// cancellable ctx supplied to polyscript.New[E] reaches the engine
// compile path. Risor's parser checks ctx; Extism's WASM compile of the
// small embedded test module completes too fast to observe a
// pre-cancelled ctx so it's covered via the HTTP-loader test below.
func TestNew_CompileCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := polyscript.New[polyscript.Risor](ctx, polyscript.FromString(`"hi"`))
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
}

// TestNew_HTTPLoaderCancelled proves the cancellable ctx actually
// reaches the HTTP loader's fetch — the precise scenario #145 set out
// to fix. The server delays the response past the ctx deadline; the
// fetch must abort with context.DeadlineExceeded rather than hanging.
func TestNew_HTTPLoaderCancelled(t *testing.T) {
	t.Parallel()

	// Stall the server long enough for the ctx timeout to fire.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	t.Cleanup(server.Close)

	ldr, err := loader.NewFromHTTP(server.URL)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	t.Cleanup(cancel)

	start := time.Now()
	_, err = polyscript.New[polyscript.Risor](ctx, polyscript.FromLoader(ldr))
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	// Sanity check: the abort happened near the deadline, not after the
	// full 2s server stall.
	assert.Less(t, time.Since(start), time.Second, "fetch should abort near the ctx deadline, not wait for the server")
}
