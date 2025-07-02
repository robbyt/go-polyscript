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

func TestFromExtismBytes(t *testing.T) {
	t.Parallel()

	t.Run("success with embedded wasmdata", func(t *testing.T) {
		// Execute
		evaluator, err := polyscript.FromExtismBytes(
			wasmdata.TestModule,
			nil,
			wasmdata.EntrypointGreet,
		)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, evaluator)
	})

	t.Run("empty bytes", func(t *testing.T) {
		// Execute
		evaluator, err := polyscript.FromExtismBytes([]byte{}, nil, wasmdata.EntrypointGreet)

		// Verify
		require.Error(t, err)
		require.Nil(t, evaluator)
	})
}

func TestFromExtismBytesWithData(t *testing.T) {
	t.Parallel()

	t.Run("success with static data", func(t *testing.T) {
		// Setup
		staticData := map[string]any{
			"version": "1.0.0",
			"config": map[string]any{
				"timeout": 30,
				"retry":   true,
			},
		}

		// Execute
		evaluator, err := polyscript.FromExtismBytesWithData(
			wasmdata.TestModule,
			staticData,
			nil,
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
		ctx, err := evaluator.AddDataToContext(t.Context(), runtimeData)
		require.NoError(t, err)

		response, err := evaluator.Eval(ctx)
		require.NoError(t, err)
		require.NotNil(t, response)
	})
}

func TestFromExtismFileLoaders(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "main.wasm")

	// Write the embedded WASM data to the temporary file
	err := os.WriteFile(wasmPath, wasmData, 0o644)
	require.NoError(t, err)

	t.Run("FromExtismFile - Valid", func(t *testing.T) {
		evaluator, err := polyscript.FromExtismFile(wasmPath, nil, wasmdata.EntrypointGreet)
		require.NoError(t, err)
		require.NotNil(t, evaluator)

		// For Extism, we need to test with correct input data
		// Create a context with the input data directly
		ctx := context.WithValue(t.Context(), constants.EvalData, map[string]any{
			"input": "Test User",
		})

		// Evaluate with the context containing input data
		result, err := evaluator.Eval(ctx)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("FromExtismFile - Invalid Path", func(t *testing.T) {
		_, err := polyscript.FromExtismFile("non-existent-file.wasm", nil, wasmdata.EntrypointGreet)
		require.Error(t, err)
	})

	t.Run("FromExtismFileWithData - Valid", func(t *testing.T) {
		staticData := map[string]any{
			"input": "Test User", // Required for WASM execution
		}
		evaluator, err := polyscript.FromExtismFileWithData(
			wasmPath,
			staticData,
			nil,
			wasmdata.EntrypointGreet,
		)
		require.NoError(t, err)
		require.NotNil(t, evaluator)
	})
}

func TestExtismDataIntegrationScenarios(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "main.wasm")

	// Write the embedded WASM data to the temporary file
	err := os.WriteFile(wasmPath, wasmData, 0o644)
	require.NoError(t, err)

	// Common test data
	staticData := map[string]any{
		"app_version": "1.0.0",
		"config": map[string]any{
			"timeout": 30,
		},
	}

	t.Run("ExtismWithData", func(t *testing.T) {
		// Create evaluator with static data that includes input
		staticDataWithInput := map[string]any{
			"input": "Test User",
		}

		extismEval, err := polyscript.FromExtismFileWithData(
			wasmPath,
			staticDataWithInput,
			nil,
			wasmdata.EntrypointGreet,
		)
		require.NoError(t, err)
		require.NotNil(t, extismEval)

		// Evaluate
		result, err := extismEval.Eval(t.Context())
		require.NoError(t, err)
		require.NotNil(t, result)

		// Check result
		resultMap, ok := result.Interface().(map[string]any)
		require.True(t, ok, "Result should be a map")
		require.Contains(t, resultMap, "greeting")
		assert.Equal(t, "Hello, Test User!", resultMap["greeting"])

		// Test evaluator with no input (should fail)
		// Create a copy of staticData without input field
		configOnlyData := map[string]any{
			"app_version": staticData["app_version"],
			"config":      staticData["config"],
		}

		extismEvalNoInput, err := polyscript.FromExtismFileWithData(
			wasmPath,
			configOnlyData,
			nil,
			wasmdata.EntrypointGreet,
		)
		require.NoError(t, err)
		require.NotNil(t, extismEvalNoInput)

		_, err = extismEvalNoInput.Eval(t.Context())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "input string is empty")
	})
}

func TestFromExtismFile(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for the WASM file
	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "main.wasm")

	// Write the embedded WASM data to the temporary file
	err := os.WriteFile(wasmPath, wasmData, 0o644)
	require.NoError(t, err)

	// Create an evaluator with file loader and static data
	evaluator, err := polyscript.FromExtismFileWithData(
		wasmPath,
		map[string]any{"input": "Test User"},
		nil,
		wasmdata.EntrypointGreet,
	)

	require.NoError(t, err)
	require.NotNil(t, evaluator)

	// Test evaluation
	result, err := evaluator.Eval(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)

	// The greet function returns a JSON with a greeting field
	resultMap, ok := result.Interface().(map[string]any)
	require.True(t, ok, "Result should be a map")
	require.Contains(t, resultMap, "greeting")
	assert.Equal(t, "Hello, Test User!", resultMap["greeting"])
}
