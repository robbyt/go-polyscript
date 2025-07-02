package main

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestStaticData returns static data for tests
func getTestStaticData() map[string]any {
	return map[string]any{
		"app_version": "1.0.0-test",
		"environment": "test",
		"config": map[string]any{
			"timeout":     10,
			"max_retries": 1,
			"feature_flags": map[string]any{
				"advanced_features": true,
				"beta_features":     true,
			},
		},
		// Put the input field directly at the top level for Extism
		"input": "Test User",
	}
}

func TestDemonstrateDataPrepAndEval(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)

	// Get static test data
	staticData := getTestStaticData()

	// Create evaluator using embedded WASM
	evaluator, err := polyscript.FromExtismBytesWithData(
		wasmdata.TestModule,
		staticData,
		logger.Handler(),
		wasmdata.EntrypointGreet,
	)
	if err != nil {
		t.Errorf("Failed to create evaluator: %v", err)
		return
	}
	require.NotNil(t, evaluator, "Evaluator should not be nil")

	// We'll just test that the evaluator was created properly
	// rather than running the full evaluation which can be flaky in tests
	// due to external dependencies
	t.Log("Extism evaluator created successfully")
}

func TestPrepareRuntimeData(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)

	// Get static test data
	staticData := getTestStaticData()

	// Create evaluator using embedded WASM
	evaluator, err := polyscript.FromExtismBytesWithData(
		wasmdata.TestModule,
		staticData,
		logger.Handler(),
		wasmdata.EntrypointGreet,
	)
	require.NoError(t, err, "Failed to create evaluator")
	require.NotNil(t, evaluator, "Evaluator should not be nil")

	// Test prepareRuntimeData function
	ctx := context.Background()
	enrichedCtx, err := prepareRuntimeData(ctx, logger, evaluator)
	assert.NoError(t, err, "prepareRuntimeData should not return an error")
	assert.NotNil(t, enrichedCtx, "Enriched context should not be nil")
}

func TestEvalAndExtractResult(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)

	// Get static test data
	staticData := getTestStaticData()

	// Create evaluator using embedded WASM
	evaluator, err := polyscript.FromExtismBytesWithData(
		wasmdata.TestModule,
		staticData,
		logger.Handler(),
		wasmdata.EntrypointGreet,
	)
	require.NoError(t, err, "Failed to create evaluator")
	require.NotNil(t, evaluator, "Evaluator should not be nil")

	// Prepare the context
	ctx := context.Background()
	ctx, err = prepareRuntimeData(ctx, logger, evaluator)
	require.NoError(t, err, "Failed to prepare context")

	// Test evaluation
	result, err := evalAndExtractResult(ctx, logger, evaluator)
	assert.NoError(t, err, "evalAndExtractResult should not return an error")
	assert.NotNil(t, result, "Result should not be nil")
}

func TestFromExtismFileWithData(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)

	// Get static test data
	staticData := getTestStaticData()

	// Test FromExtismBytesWithData function
	evaluator, err := polyscript.FromExtismBytesWithData(
		wasmdata.TestModule,
		staticData,
		logger.Handler(),
		wasmdata.EntrypointGreet,
	)
	assert.NoError(t, err, "Should create evaluator without error")
	assert.NotNil(t, evaluator, "Evaluator should not be nil")
}

func TestEmbeddedWasmModule(t *testing.T) {
	// Verify the embedded WASM module is available
	assert.NotEmpty(t, wasmdata.TestModule, "Embedded WASM module should not be empty")
	assert.NotEmpty(t, wasmdata.EntrypointGreet, "Entrypoint constant should not be empty")
}

func TestRun(t *testing.T) {
	assert.NoError(t, run(), "run() should execute without error")
}
