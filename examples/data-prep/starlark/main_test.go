package main

import (
	"context"
	"log/slog"
	"testing"

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
	}
}

func TestRun(t *testing.T) {
	// This is a simple test of the run function
	err := run()
	assert.NoError(t, err, "run() should execute without error")
}

func TestCreateStarlarkEvaluator(t *testing.T) {
	logger := slog.Default()
	staticData := getTestStaticData()

	// Test creating evaluator
	evaluator, err := createStarlarkEvaluator(logger, starlarkScript, staticData)
	require.NoError(t, err, "Should create evaluator without error")

	// Verify the evaluator is functional
	ctx := context.Background()
	evalResult, err := evaluator.Eval(ctx)
	require.NoError(t, err, "Simple evaluation should succeed")
	require.NotNil(t, evalResult, "Evaluation result should not be nil")
}

func TestPrepareRuntimeData(t *testing.T) {
	logger := slog.Default()
	staticData := getTestStaticData()

	// Create evaluator
	evaluator, err := createStarlarkEvaluator(logger, starlarkScript, staticData)
	require.NoError(t, err, "Failed to create evaluator")

	// Test prepareRuntimeData function
	ctx := context.Background()
	enrichedCtx, err := prepareRuntimeData(ctx, logger, evaluator)
	assert.NoError(t, err, "prepareRuntimeData should not return an error")
	assert.NotNil(t, enrichedCtx, "Enriched context should not be nil")
}

func TestEvalAndExtractResult(t *testing.T) {
	logger := slog.Default()
	staticData := getTestStaticData()

	// Create evaluator
	evaluator, err := createStarlarkEvaluator(logger, starlarkScript, staticData)
	require.NoError(t, err, "Failed to create evaluator")

	// Prepare data first
	ctx := context.Background()
	preparedCtx, err := prepareRuntimeData(ctx, logger, evaluator)
	require.NoError(t, err, "Failed to prepare context")

	// Test evaluation
	result, err := evalAndExtractResult(preparedCtx, logger, evaluator)
	assert.NoError(t, err, "evalAndExtractResult should not return an error")
	assert.NotNil(t, result, "Result should not be nil")

	// Check basic result fields
	assert.Contains(t, result, "greeting", "Result should contain a greeting")
	assert.Contains(t, result, "app_info", "Result should contain app info")

	// Verify app info contains data from static provider
	appInfo, hasAppInfo := result["app_info"].(map[string]any)
	if assert.True(t, hasAppInfo, "app_info should be a map") {
		assert.Equal(t, "1.0.0-test", appInfo["version"], "Should have correct app version")
		assert.Equal(t, "test", appInfo["environment"], "Should have test environment")
	}
}

// TestFullExecution tests the entire execution flow as an integration test
func TestFullExecution(t *testing.T) {
	logger := slog.Default()
	staticData := getTestStaticData()

	// Create evaluator with static data
	evaluator, err := createStarlarkEvaluator(logger, starlarkScript, staticData)
	require.NoError(t, err, "Failed to create evaluator")

	// Prepare runtime data
	ctx := context.Background()
	preparedCtx, err := prepareRuntimeData(ctx, logger, evaluator)
	require.NoError(t, err, "Failed to prepare runtime data")

	// Execute the script
	result, err := evalAndExtractResult(preparedCtx, logger, evaluator)
	require.NoError(t, err, "Failed to evaluate script")
	require.NotNil(t, result, "Result should not be nil")

	// Verify result has data from both static and dynamic providers
	assert.Contains(t, result, "greeting", "Result should have greeting from dynamic data")
	assert.Contains(t, result, "user_id", "Result should have user_id from dynamic data")
	assert.Contains(t, result, "app_info", "Result should have app_info from static data")
	assert.Contains(t, result, "request_info", "Result should have request_info from HTTP request")
}
