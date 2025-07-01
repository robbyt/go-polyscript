package polyscript

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/stretchr/testify/require"
)

func TestFromExtismBytes(t *testing.T) {
	t.Parallel()

	t.Run("success with embedded wasmdata", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)

		// Execute
		evaluator, err := FromExtismBytes(wasmdata.TestModule, handler, wasmdata.EntrypointGreet)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evaluator)
	})

	t.Run("empty bytes", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)

		// Execute
		evaluator, err := FromExtismBytes([]byte{}, handler, wasmdata.EntrypointGreet)

		// Verify
		require.Error(t, err)
		require.Nil(t, evaluator)
	})
}

func TestFromExtismBytesWithData(t *testing.T) {
	t.Parallel()

	t.Run("success with static data", func(t *testing.T) {
		// Setup
		handler := slog.NewTextHandler(os.Stdout, nil)
		staticData := map[string]any{
			"version": "1.0.0",
			"config": map[string]any{
				"timeout": 30,
				"retry":   true,
			},
		}

		// Execute
		evaluator, err := FromExtismBytesWithData(
			wasmdata.TestModule,
			staticData,
			handler,
			wasmdata.EntrypointGreet,
		)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evaluator)

		// Test that we can actually run the evaluator with the embedded WASM
		// Add runtime data with required "input" field
		runtimeData := map[string]any{
			"input": "test user",
		}
		ctx, err := evaluator.AddDataToContext(context.Background(), runtimeData)
		require.NoError(t, err)

		response, err := evaluator.Eval(ctx)
		require.NoError(t, err)
		require.NotNil(t, response)
	})
}
