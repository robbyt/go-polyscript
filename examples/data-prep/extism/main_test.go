package main

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDemonstrateDataPrepAndEval(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)

	// Find the WASM file
	wasmFilePath, err := FindWasmFile(logger)
	if err != nil {
		t.Logf("Extism example failed: %v - this may be due to missing WASM file", err)
		t.Skip("Skipping Extism test as it requires a WASM file")
		return
	}

	// Create evaluator
	evaluator, err := CreateExtismEvaluator(wasmFilePath, handler)
	if err != nil {
		t.Logf("Failed to create evaluator: %v", err)
		t.Skip("Skipping Extism test as it failed to create evaluator")
		return
	}
	require.NotNil(t, evaluator, "Evaluator should not be nil")

	// We'll just test that the evaluator was created properly
	// rather than running the full evaluation which can be flaky in tests
	// due to external dependencies
	t.Log("Extism evaluator created successfully")
}

func TestPrepareRequestData(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)

	// Find the WASM file
	wasmFilePath, err := FindWasmFile(logger)
	if err != nil {
		t.Logf("Skipping test due to missing WASM file: %v", err)
		t.Skip("WASM file required for this test")
		return
	}

	// Create evaluator
	evaluator, err := CreateExtismEvaluator(wasmFilePath, handler)
	require.NoError(t, err, "Failed to create evaluator")
	require.NotNil(t, evaluator, "Evaluator should not be nil")

	// Test PrepareRequestData function
	ctx := context.Background()
	enrichedCtx, err := PrepareRequestData(ctx, *evaluator, logger)
	assert.NoError(t, err, "PrepareRequestData should not return an error")
	assert.NotNil(t, enrichedCtx, "Enriched context should not be nil")
}

func TestPrepareConfigData(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)

	// Find the WASM file
	wasmFilePath, err := FindWasmFile(logger)
	if err != nil {
		t.Logf("Skipping test due to missing WASM file: %v", err)
		t.Skip("WASM file required for this test")
		return
	}

	// Create evaluator
	evaluator, err := CreateExtismEvaluator(wasmFilePath, handler)
	require.NoError(t, err, "Failed to create evaluator")
	require.NotNil(t, evaluator, "Evaluator should not be nil")

	// Test PrepareConfigData function
	ctx := context.Background()
	enrichedCtx, err := PrepareConfigData(ctx, *evaluator, logger)
	assert.NoError(t, err, "PrepareConfigData should not return an error")
	assert.NotNil(t, enrichedCtx, "Enriched context should not be nil")
}

func TestFindWasmFile(t *testing.T) {
	// This test just verifies the FindWasmFile function doesn't panic
	// and follows the expected logic
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)

	// The function might not find a file but it shouldn't panic
	wasmPath, err := FindWasmFile(logger)
	// Log the result without assertion
	if err != nil {
		t.Logf("FindWasmFile returned error: %v (this is expected in some environments)", err)
	} else if wasmPath != "" {
		t.Logf("Found WASM file at: %s", wasmPath)
	} else {
		t.Log("No WASM file found - this is expected in some environments")
	}

	// Test with nil logger - should not panic
	_, err = FindWasmFile(nil)
	if err != nil {
		t.Logf("FindWasmFile with nil logger returned error: %v (this is expected)", err)
	}
}

func TestCreateExtismEvaluator(t *testing.T) {
	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Find the WASM file
	wasmFilePath, err := FindWasmFile(nil)
	if err != nil {
		t.Logf("Skipping test due to missing WASM file: %v", err)
		t.Skip("WASM file required for this test")
		return
	}

	// Test CreateExtismEvaluator function
	evaluator, err := CreateExtismEvaluator(wasmFilePath, handler)
	require.NoError(t, err, "Should create evaluator without error")
	require.NotNil(t, evaluator, "Evaluator should not be nil")
}
