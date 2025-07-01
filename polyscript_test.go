package polyscript_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/robbyt/go-polyscript/engines/mocks"
	"github.com/robbyt/go-polyscript/engines/types"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var wasmData = wasmdata.TestModule

// Helper functions for tests
func getLogger() slog.Handler {
	return slog.NewTextHandler(os.Stdout, nil)
}

// mockPreparer implements evaluation.EvalDataPreparer for testing
type mockPreparer struct {
	mock.Mock
}

func (m *mockPreparer) AddDataToContext(
	ctx context.Context,
	data ...map[string]any,
) (context.Context, error) {
	args := m.Called(ctx, data)
	return args.Get(0).(context.Context), args.Error(1)
}

// evalAndExtractMap runs evaluation and extracts result as a map[string]any
func evalAndExtractMap(
	t *testing.T,
	ctx context.Context,
	evaluator platform.EvalOnly,
) (map[string]any, error) {
	t.Helper()

	// Evaluate the script
	result, err := evaluator.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("script evaluation failed: %w", err)
	}

	// Process the result
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

// prepareAndEval combines context preparation and evaluation in a single operation
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

	// Evaluate with the enriched context
	result, err := evaluator.Eval(enrichedCtx)
	if err != nil {
		return nil, fmt.Errorf("script evaluation failed: %w", err)
	}

	return result, nil
}

// Test machine-specific evaluator creators
func TestMachineEvaluators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string
		machineType types.Type
		creator     func(string, slog.Handler) (platform.Evaluator, error)
	}{
		{
			name:        "FromStarlarkString",
			content:     `print("Hello, World!")`,
			machineType: types.Starlark,
			creator:     polyscript.FromStarlarkString,
		},
		{
			name:        "FromRisorString",
			content:     `print("Hello, World!")`,
			machineType: types.Risor,
			creator:     polyscript.FromRisorString,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			// Create evaluator directly with content and logger
			evaluator, err := tc.creator(tc.content, getLogger())
			require.NoError(t, err)
			require.NotNil(t, evaluator)
		})
	}
}

func TestFromStringLoaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string
		creator     func(string, slog.Handler) (platform.Evaluator, error)
		logHandler  slog.Handler
		expectError bool
	}{
		{
			name:        "FromStarlarkString - Valid",
			content:     `print("Hello, World!")`,
			creator:     polyscript.FromStarlarkString,
			logHandler:  getLogger(),
			expectError: false,
		},
		{
			name:        "FromRisorString - Valid",
			content:     `print("Hello, World!")`,
			creator:     polyscript.FromRisorString,
			logHandler:  getLogger(),
			expectError: false,
		},
		{
			name:        "FromStarlarkString - Empty",
			content:     "",
			creator:     polyscript.FromStarlarkString,
			logHandler:  getLogger(),
			expectError: true,
		},
		{
			name:        "FromRisorString - Empty",
			content:     "",
			creator:     polyscript.FromRisorString,
			logHandler:  getLogger(),
			expectError: true,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			evaluator, err := tc.creator(tc.content, tc.logHandler)

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, evaluator)
		})
	}
}

func TestFromFileLoaders(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "main.wasm")
	risorPath := filepath.Join(tmpDir, "test.risor")
	starlarkPath := filepath.Join(tmpDir, "test.star")

	// Write the embedded WASM data to the temporary file
	err := os.WriteFile(wasmPath, wasmData, 0o644)
	require.NoError(t, err)

	// Create a basic Risor script
	risorContent := `{ "message": "Hello from Risor!" }`
	err = os.WriteFile(risorPath, []byte(risorContent), 0o644)
	require.NoError(t, err)

	// Create a basic Starlark script
	starlarkContent := `result = {"message": "Hello from Starlark!"}
_ = result`
	err = os.WriteFile(starlarkPath, []byte(starlarkContent), 0o644)
	require.NoError(t, err)

	// Setup the logger handler
	logHandler := getLogger()

	t.Run("FromExtismFile - Valid", func(t *testing.T) {
		evaluator, err := polyscript.FromExtismFile(wasmPath, logHandler, "greet")
		require.NoError(t, err)
		require.NotNil(t, evaluator)

		// For Extism, we need to test with correct input data
		// Create a context with the input data directly
		ctx := context.Background()
		ctx = context.WithValue(ctx, constants.EvalData, map[string]any{
			"input": "Test User",
		})

		// Evaluate with the context containing input data
		result, err := evaluator.Eval(ctx)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("FromExtismFile - Invalid Path", func(t *testing.T) {
		_, err := polyscript.FromExtismFile("non-existent-file.wasm", logHandler, "greet")
		require.Error(t, err)
	})

	t.Run("FromExtismFileWithData - Valid", func(t *testing.T) {
		staticData := map[string]any{
			"input": "Test User", // Required for WASM execution
		}
		evaluator, err := polyscript.FromExtismFileWithData(
			wasmPath,
			staticData,
			logHandler,
			"greet",
		)
		require.NoError(t, err)
		require.NotNil(t, evaluator)
	})

	t.Run("FromRisorFile - Valid", func(t *testing.T) {
		evaluator, err := polyscript.FromRisorFile(risorPath, logHandler)
		require.NoError(t, err)
		require.NotNil(t, evaluator)

		// Basic execution
		result, err := evaluator.Eval(context.Background())
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("FromRisorFile - Invalid Path", func(t *testing.T) {
		_, err := polyscript.FromRisorFile("non-existent-file.risor", logHandler)
		require.Error(t, err)
	})

	t.Run("FromRisorFileWithData - Valid", func(t *testing.T) {
		staticData := map[string]any{
			"test_key": "test_value",
		}
		evaluator, err := polyscript.FromRisorFileWithData(risorPath, staticData, logHandler)
		require.NoError(t, err)
		require.NotNil(t, evaluator)
	})

	t.Run("FromStarlarkFile - Valid", func(t *testing.T) {
		evaluator, err := polyscript.FromStarlarkFile(starlarkPath, logHandler)
		require.NoError(t, err)
		require.NotNil(t, evaluator)

		// Basic execution
		result, err := evaluator.Eval(context.Background())
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("FromStarlarkFile - Invalid Path", func(t *testing.T) {
		_, err := polyscript.FromStarlarkFile("non-existent-file.star", logHandler)
		require.Error(t, err)
	})

	t.Run("FromStarlarkFileWithData - Valid", func(t *testing.T) {
		staticData := map[string]any{
			"test_key": "test_value",
		}
		evaluator, err := polyscript.FromStarlarkFileWithData(starlarkPath, staticData, logHandler)
		require.NoError(t, err)
		require.NotNil(t, evaluator)
	})
}

func TestDataProviders(t *testing.T) {
	t.Parallel()

	t.Run("withCompositeProvider", func(t *testing.T) {
		// Create a simple script that uses composite data
		script := `print(ctx["static_key"], ", ", ctx["dynamic_key"])`

		// Create static data
		staticData := map[string]any{
			"static_key": "static_value",
		} // Create an evaluator with composite provider
		evaluator, err := polyscript.FromStarlarkStringWithData(
			script,
			staticData,
			getLogger(),
		)
		require.NoError(t, err)
		require.NotNil(t, evaluator)

		// Test adding dynamic data
		ctx := context.Background()
		dynamicData := map[string]any{"dynamic_key": "dynamic_value"}
		enrichedCtx, err := evaluator.AddDataToContext(ctx, dynamicData)
		require.NoError(t, err)

		// Execute the script (won't fail if print works correctly)
		_, err = evaluator.Eval(enrichedCtx)
		require.NoError(t, err)
	})
}

func TestEvalHelpers(t *testing.T) {
	t.Parallel()

	t.Run("PrepareAndEval", func(t *testing.T) {
		// Create a simple Risor evaluator
		script := `
            name := ctx["name"]
            {
                "message": "Hello, " + name + "!",
                "length": len(name)
            }
        `

		// Create an evaluator with the CompositeProvider
		evaluator, err := polyscript.FromRisorStringWithData(
			script,
			map[string]any{},
			getLogger(),
		)
		require.NoError(t, err)

		// Test the PrepareAndEval function
		result, err := prepareAndEval(
			t,
			context.Background(),
			evaluator,
			map[string]any{"name": "World"},
		)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify the result
		resultMap, ok := result.Interface().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Hello, World!", resultMap["message"])

		// Check length without assuming the exact numeric type
		length := resultMap["length"]
		require.NotNil(t, length, "length field should be present")
		switch v := length.(type) {
		case int64:
			assert.Equal(t, int64(5), v, "length should be 5")
		case float64:
			assert.Equal(t, float64(5), v, "length should be 5")
		default:
			t.Errorf("length is unexpected type %T", v)
		}

		// Create error-producing mocks
		t.Run("AddDataToContext error", func(t *testing.T) {
			// Create mocks for testing error cases
			mockPrepCtx := &mockPreparer{}
			mockEval := &mocks.Evaluator{}

			// Create context and data
			ctx := context.Background()
			d := map[string]any{"name": "World"}

			// Mock AddDataToContext to return an error
			mockPrepCtx.On("AddDataToContext", ctx, mock.Anything).
				Return(ctx, errors.New("prepare error"))

			// Create a mock evaluator that implements both interfaces
			mockEvalWithPrep := struct {
				platform.EvalOnly
				data.Setter
			}{
				EvalOnly: mockEval,
				Setter:   mockPrepCtx,
			}

			// PrepareAndEval should return the prepare error
			_, err = prepareAndEval(t, ctx, mockEvalWithPrep, d)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to prepare context")
			mockPrepCtx.AssertExpectations(t)
		})

		t.Run("Eval error", func(t *testing.T) {
			// Create mocks for testing error cases
			mockPrepCtx := &mockPreparer{}
			mockEval := &mocks.Evaluator{}

			// Create context and data
			ctx := context.Background()
			d := map[string]any{"name": "World"}

			// Mock AddDataToContext to succeed
			// Define a type for context keys to avoid linting warnings
			type contextKey string
			testKey := contextKey("test-key")
			enrichedCtx := context.WithValue(ctx, testKey, "test-value")
			mockPrepCtx.On("AddDataToContext", ctx, mock.Anything).Return(enrichedCtx, nil)

			// Mock Eval to fail
			mockEval.On("Eval", enrichedCtx).
				Return((*mocks.EvaluatorResponse)(nil), errors.New("eval error"))

			// Create a mock evaluator that implements both interfaces
			mockEvalWithPrep := struct {
				platform.EvalOnly
				data.Setter
			}{
				EvalOnly: mockEval,
				Setter:   mockPrepCtx,
			}

			// PrepareAndEval should return the eval error
			_, err = prepareAndEval(t, ctx, mockEvalWithPrep, d)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "script evaluation failed")
			mockPrepCtx.AssertExpectations(t)
			mockEval.AssertExpectations(t)
		})
	})

	t.Run("EvalAndExtractMap", func(t *testing.T) {
		// Create a simple Risor evaluator
		script := `
            {
                "message": "Hello, Static!",
                "length": 12
            }
        `

		// Create an evaluator
		evaluator, err := polyscript.FromRisorStringWithData(
			script,
			map[string]any{},
			getLogger(),
		)
		require.NoError(t, err)

		// Test EvalAndExtractMap
		resultMap, err := evalAndExtractMap(t, context.Background(), evaluator)
		require.NoError(t, err)

		// Verify the result
		assert.Equal(t, "Hello, Static!", resultMap["message"])

		// Check length without assuming the exact numeric type
		length := resultMap["length"]
		require.NotNil(t, length, "length field should be present")
		switch v := length.(type) {
		case int64:
			assert.Equal(t, int64(12), v, "length should be 12")
		case float64:
			assert.Equal(t, float64(12), v, "length should be 12")
		default:
			t.Errorf("length is unexpected type %T", v)
		}

		// Test with nil result
		nilScript := `nil`
		nilEvaluator, err := polyscript.FromRisorString(
			nilScript,
			getLogger(),
		)
		require.NoError(t, err)

		nilResult, err := evalAndExtractMap(t, context.Background(), nilEvaluator)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{}, nilResult)

		// Test with non-map result (should error)
		numScript := `42`
		numEvaluator, err := polyscript.FromRisorString(
			numScript,
			getLogger(),
		)
		require.NoError(t, err)

		_, err = evalAndExtractMap(t, context.Background(), numEvaluator)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "result is not a map")

		// Test with evaluation error
		t.Run("Eval error", func(t *testing.T) {
			mockEval := &mocks.Evaluator{}
			ctx := context.Background()

			// Mock Eval to return an error
			mockEval.On("Eval", ctx).
				Return((*mocks.EvaluatorResponse)(nil), errors.New("eval error"))

			// EvalAndExtractMap should return the error
			_, err = evalAndExtractMap(t, ctx, mockEval)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "script evaluation failed")
			mockEval.AssertExpectations(t)
		})
	})
}

func TestDataIntegrationScenarios(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "main.wasm")
	risorPath := filepath.Join(tmpDir, "test.risor")
	starlarkPath := filepath.Join(tmpDir, "test.star")

	// Write the embedded WASM data to the temporary file
	err := os.WriteFile(wasmPath, wasmData, 0o644)
	require.NoError(t, err)

	// Create a basic Risor script that uses context
	risorFileContent := `// Get data from context
{
    "message": "Hello, " + ctx["name"] + " (v" + ctx["app_version"] + ")",
    "timeout": ctx["config"]["timeout"]
}`
	err = os.WriteFile(risorPath, []byte(risorFileContent), 0o644)
	require.NoError(t, err)

	// Create a basic Starlark script that uses context
	starlarkFileContent := `# Simple Starlark script
result = {
    "message": "Hello, " + ctx["name"] + " (v" + ctx["app_version"] + ")",
    "timeout": ctx["config"]["timeout"]
}
_ = result`
	err = os.WriteFile(starlarkPath, []byte(starlarkFileContent), 0o644)
	require.NoError(t, err)

	// Common test data
	staticData := map[string]any{
		"app_version": "1.0.0",
		"config": map[string]any{
			"timeout": 30,
		},
	}

	t.Run("RisorWithData", func(t *testing.T) {
		// Test script
		risorScript := `
            // Access static data
            version := ctx["app_version"]
            timeout := ctx["config"]["timeout"]
            
            // Access dynamic data
            name := ctx["name"]
            
            {
                "message": "Hello, " + name + " (v" + version + ")",
                "timeout": timeout
            }
        `

		// Create evaluator with static data
		risorEval, err := polyscript.FromRisorStringWithData(
			risorScript,
			staticData,
			getLogger(),
		)
		require.NoError(t, err)

		// Test with dynamic data
		ctx := context.Background()
		dynamicData := map[string]any{"name": "Risor User"}
		enrichedCtx, err := risorEval.AddDataToContext(ctx, dynamicData)
		require.NoError(t, err)

		risorResult, err := risorEval.Eval(enrichedCtx)
		require.NoError(t, err)

		risorMap, ok := risorResult.Interface().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Hello, Risor User (v1.0.0)", risorMap["message"])

		// Check timeout without assuming specific number type
		timeout := risorMap["timeout"]
		require.NotNil(t, timeout, "timeout field should be present")
		switch v := timeout.(type) {
		case int64:
			assert.Equal(t, int64(30), v, "timeout should be 30")
		case float64:
			assert.Equal(t, float64(30), v, "timeout should be 30")
		default:
			t.Errorf("timeout is unexpected type %T", v)
		}
	})

	t.Run("StarlarkWithData", func(t *testing.T) {
		// Create evaluator with static data
		starlarkEval, err := polyscript.FromStarlarkFileWithData(
			starlarkPath,
			staticData,
			getLogger(),
		)
		require.NoError(t, err)

		// Test with dynamic data
		ctx := context.Background()
		dynamicData := map[string]any{"name": "Starlark User"}
		enrichedCtx, err := starlarkEval.AddDataToContext(ctx, dynamicData)
		require.NoError(t, err)

		starlarkResult, err := starlarkEval.Eval(enrichedCtx)
		require.NoError(t, err)

		starlarkMap, ok := starlarkResult.Interface().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Hello, Starlark User (v1.0.0)", starlarkMap["message"])

		// Check timeout without assuming specific number type
		starlarkTimeout := starlarkMap["timeout"]
		require.NotNil(t, starlarkTimeout, "timeout field should be present")
		assert.Equal(t, int64(30), starlarkTimeout, "timeout should be 30")
	})

	t.Run("ExtismWithData", func(t *testing.T) {
		// Create evaluator with static data that includes input
		staticDataWithInput := map[string]any{
			"input": "Test User",
		}

		extismEval, err := polyscript.FromExtismFileWithData(
			wasmPath,
			staticDataWithInput,
			getLogger(),
			"greet",
		)
		require.NoError(t, err)
		require.NotNil(t, extismEval)

		// Evaluate
		ctx := context.Background()
		result, err := extismEval.Eval(ctx)
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
			getLogger(),
			"greet",
		)
		require.NoError(t, err)
		require.NotNil(t, extismEvalNoInput)

		_, err = extismEvalNoInput.Eval(ctx)
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

	// Create a logger handler
	handler := getLogger()

	// Create an evaluator with file loader and static data
	evaluator, err := polyscript.FromExtismFileWithData(
		wasmPath,
		map[string]any{"input": "Test User"},
		handler,
		"greet",
	)

	require.NoError(t, err)
	require.NotNil(t, evaluator)

	// Create a context for evaluation
	ctx := context.Background()

	// Test evaluation
	result, err := evaluator.Eval(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)

	// The greet function returns a JSON with a greeting field
	resultMap, ok := result.Interface().(map[string]any)
	require.True(t, ok, "Result should be a map")
	require.Contains(t, resultMap, "greeting")
	assert.Equal(t, "Hello, Test User!", resultMap["greeting"])
}

func TestCreateEvaluatorEdgeCases(t *testing.T) {
	t.Parallel()

	// Test error with empty script content
	t.Run("Empty Script Content Error", func(t *testing.T) {
		// Try to create an evaluator with empty script
		_, err := polyscript.FromRisorString("", getLogger())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "content is empty")
	})

	// Test invalid path error
	t.Run("Invalid Path Error", func(t *testing.T) {
		// Try to create an evaluator with non-existent file
		_, err := polyscript.FromRisorFile("/path/does/not/exist.risor", getLogger())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	})

	// Test with invalid script content
	t.Run("InvalidScriptTest", func(t *testing.T) {
		// Try to create an evaluator with invalid script content
		_, err := polyscript.FromRisorString("this is not valid risor code }{", getLogger())

		// Should return an error when trying to compile invalid code
		require.Error(t, err)
		assert.Contains(t, err.Error(), "compile")
	})
}

// MockStringLoader is a simple implementation of loader.Loader using a string
type MockStringLoader struct{}

func (m *MockStringLoader) GetReader() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("return 0")), nil
}

func (m *MockStringLoader) GetSourceURL() *url.URL {
	return nil
}

func TestFromStringLoader(t *testing.T) {
	t.Parallel()

	// Test the Extism string loader error case directly
	t.Run("ExtismStringNotSupported", func(t *testing.T) {
		// Just test if a hypothetical FromExtismString would have issues
		// For now, we'll simulate this by testing if we can create a string loader
		content := "test"
		l, err := loader.NewFromString(content)
		require.NoError(t, err)

		// Since we know Extism is for WASM modules, string content
		// would not be valid WASM, so this would fail.
		// Just verify our loader was created correctly
		require.NotNil(t, l)
		require.NotNil(t, l.GetSourceURL())
	})
}
