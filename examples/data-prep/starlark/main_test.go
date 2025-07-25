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

	// Verify actual dynamic data is used, not fallback values
	greeting := result["greeting"]
	require.IsType(t, "", greeting, "Greeting should be a string")
	assert.Equal(t,
		"Hello, World!", greeting,
		"Should use actual name 'World', not fallback 'Default'")

	userID := result["user_id"]
	require.IsType(t, "", userID, "user_id should be a string")
	assert.Equal(t, "user-123", userID, "Should use actual user_id, not fallback 'unknown'")

	timestamp := result["timestamp"]
	require.IsType(t, "", timestamp, "timestamp should be a string")
	assert.NotEqual(t, "Unknown", timestamp, "Should use actual timestamp, not fallback 'Unknown'")

	// Verify app info contains data from static provider
	appInfo := result["app_info"]
	require.IsType(t, map[string]any{}, appInfo, "app_info should be a map")
	appInfoMap := appInfo.(map[string]any)
	assert.Equal(t, "1.0.0-test", appInfoMap["version"], "Should have correct app version")
	assert.Equal(t, "test", appInfoMap["environment"], "Should have test environment")
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

	// Verify the dynamic data is returned, not the fallback values
	greeting := result["greeting"]
	require.IsType(t, "", greeting, "Greeting should be a string")
	assert.Equal(
		t,
		"Hello, World!",
		greeting,
		"Should use actual name 'World', not fallback 'Default'",
	)

	userID := result["user_id"]
	require.IsType(t, "", userID, "user_id should be a string")
	assert.Equal(t, "user-123", userID, "Should use actual user_id, not fallback 'unknown'")

	timestamp := result["timestamp"]
	require.IsType(t, "", timestamp, "timestamp should be a string")
	assert.NotEqual(t, "Unknown", timestamp, "Should use actual timestamp, not fallback 'Unknown'")

	message := result["message"]
	require.IsType(t, "", message, "message should be a string")
	assert.Contains(
		t,
		message,
		"admin",
		"Should use actual user role 'admin', not fallback 'guest'",
	)
	assert.NotContains(t, message, "guest", "Should not contain fallback role 'guest'")
}
