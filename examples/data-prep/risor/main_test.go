package main

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create Risor evaluator - makes testing easier
func createRisorEvalHelper(t *testing.T, logger *slog.Logger) (RisorEvaluator, error) {
	t.Helper()

	// Get static test data
	staticData := getTestStaticData()

	// Create evaluator with the script and static data
	return createRisorEvaluator(logger, risorScript, staticData)
}

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

func TestCreateRisorEvaluator(t *testing.T) {
	// Test creating evaluator
	evaluator, err := createRisorEvalHelper(t, slog.Default())
	require.NoError(t, err, "Should create evaluator without error")
	require.NotNil(t, evaluator, "Evaluator should not be nil")
}

func TestPrepareRuntimeData(t *testing.T) {
	logger := slog.Default()

	// Create evaluator
	evaluator, err := createRisorEvalHelper(t, logger)
	require.NoError(t, err, "Failed to create evaluator")

	// Test prepareRuntimeData function
	ctx := context.Background()
	enrichedCtx, err := prepareRuntimeData(ctx, logger, evaluator)
	assert.NoError(t, err, "prepareRuntimeData should not return an error")
	assert.NotNil(t, enrichedCtx, "Enriched context should not be nil")
}

func TestEvalAndExtractResult(t *testing.T) {
	logger := slog.Default()

	// Create evaluator
	evaluator, err := createRisorEvalHelper(t, logger)
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
	assert.Contains(t,
		result, "greeting",
		"Result should contain a greeting")

	// Verify actual dynamic data is returned, not the fallback values
	greeting := result["greeting"]
	require.IsType(t, "", greeting, "Greeting should be a string")
	assert.Equal(t,
		"Hello, World!", greeting,
		"Should use actual name 'World', not fallback 'Default'")

	userID := result["user_id"]
	require.IsType(t, "", userID, "user_id should be a string")
	assert.Equal(t,
		"user-123", userID,
		"Should use actual user_id, not fallback 'unknown'")

	timestamp := result["timestamp"]
	require.IsType(t,
		"", timestamp,
		"timestamp should be a string")
	assert.NotEqual(t,
		"Unknown", timestamp,
		"Should use actual timestamp, not fallback 'Unknown'")
}

// TestFullExecution tests the entire execution flow as an integration test
func TestFullExecution(t *testing.T) {
	// This test mirrors the functionality in the run() function
	logger := slog.Default()

	// Get test static data
	staticData := getTestStaticData()

	// Create evaluator
	evaluator, err := createRisorEvaluator(logger, risorScript, staticData)
	require.NoError(t, err, "Failed to create evaluator")

	// Prepare runtime data
	ctx := context.Background()
	preparedCtx, err := prepareRuntimeData(ctx, logger, evaluator)
	require.NoError(t, err, "Failed to prepare runtime data")

	// Execute the script
	result, err := evalAndExtractResult(preparedCtx, logger, evaluator)
	require.NoError(t, err, "Failed to evaluate script")
	require.NotNil(t, result, "Result should not be nil")

	// Verify result has expected fields
	assert.Contains(t, result, "greeting", "Result should have greeting")
	assert.Contains(t, result, "user_id", "Result should have user_id")
	assert.Contains(t, result, "app_info", "Result should have app_info")

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
