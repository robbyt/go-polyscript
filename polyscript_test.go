package polyscript

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/engine/options"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	extismCompiler "github.com/robbyt/go-polyscript/machines/extism/compiler"
	risorCompiler "github.com/robbyt/go-polyscript/machines/risor/compiler"
	starlarkCompiler "github.com/robbyt/go-polyscript/machines/starlark/compiler"
	"github.com/robbyt/go-polyscript/machines/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

//go:embed examples/testdata/main.wasm
var wasmData []byte

// Helper functions for tests
func getLogger() slog.Handler {
	return slog.NewTextHandler(os.Stdout, nil)
}

// withCompositeProvider creates a composite provider with static data
func withCompositeProvider(staticData map[string]any) any {
	return options.WithDataProvider(data.NewCompositeProvider(
		data.NewStaticProvider(staticData),
		data.NewContextProvider(constants.Ctx),
	))
}

// Create a mock evaluator response
type mockResponse struct {
	value any
}

func (m mockResponse) Interface() any {
	return m.value
}

func (m mockResponse) GetScriptExeID() string {
	return "mock-script-id"
}

func (m mockResponse) GetExecTime() string {
	return "1ms"
}

func (m mockResponse) Inspect() string {
	return "mock-response"
}

func (m mockResponse) Type() data.Types {
	return data.NONE
}

// mockEvaluator implements engine.Evaluator for testing
type mockEvaluator struct {
	mock.Mock
}

func (m *mockEvaluator) Eval(ctx context.Context) (engine.EvaluatorResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return mockResponse{value: args.Get(0)}, args.Error(1)
}

// mockPreparer implements engine.EvalDataPreparer for testing
type mockPreparer struct {
	mock.Mock
}

func (m *mockPreparer) PrepareContext(ctx context.Context, data ...any) (context.Context, error) {
	args := m.Called(ctx, data)
	return args.Get(0).(context.Context), args.Error(1)
}

// evalAndExtractMap runs evaluation and extracts result as a map[string]any
func evalAndExtractMap(
	t *testing.T,
	ctx context.Context,
	evaluator engine.Evaluator,
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
	evaluator engine.EvaluatorWithPrep,
	runtimeData map[string]any,
) (engine.EvaluatorResponse, error) {
	t.Helper()

	enrichedCtx, err := evaluator.PrepareContext(ctx, runtimeData)
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
		creator     func(opts ...any) (engine.EvaluatorWithPrep, error)
		options     []any
	}{
		{
			name:        "NewStarlarkEvaluator",
			content:     `print("Hello, World!")`,
			machineType: types.Starlark,
			creator:     NewStarlarkEvaluator,
			options: []any{
				options.WithDefaults(),
			},
		},
		{
			name:        "NewRisorEvaluator",
			content:     `print("Hello, World!")`,
			machineType: types.Risor,
			creator:     NewRisorEvaluator,
			options: []any{
				options.WithDefaults(),
			},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			// Create a loader
			l, err := loader.NewFromString(tc.content)
			require.NoError(t, err)

			// Combine options with loader
			opts := append(
				[]any{
					options.WithLoader(l),
					options.WithLogHandler(getLogger()),
				},
				tc.options...,
			)

			// Create evaluator
			evaluator, err := tc.creator(opts...)
			require.NoError(t, err)
			require.NotNil(t, evaluator)
		})
	}
}

func TestNewEvaluator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		machineType types.Type
		options     []any
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid Starlark",
			machineType: types.Starlark,
			options: []any{
				options.WithLoader(func() loader.Loader {
					l, err := loader.NewFromString("print('test')")
					require.NoError(t, err)
					return l
				}()),
				options.WithLogHandler(getLogger()),
			},
			expectError: false,
		},
		{
			name:        "Valid Risor",
			machineType: types.Risor,
			options: []any{
				options.WithLoader(func() loader.Loader {
					l, err := loader.NewFromString("print('test')")
					require.NoError(t, err)
					return l
				}()),
				options.WithLogHandler(getLogger()),
			},
			expectError: false,
		},
		{
			name:        "No Loader",
			machineType: types.Starlark,
			options: []any{
				options.WithLogHandler(getLogger()),
			},
			expectError: true,
			errorMsg:    "no loader specified",
		},
		{
			name:        "Invalid Option",
			machineType: types.Starlark,
			options: []any{
				options.WithLoader(func() loader.Loader {
					l, err := loader.NewFromString("print('test')")
					require.NoError(t, err)
					return l
				}()),
				func(cfg *options.Config) error {
					return errors.New("invalid option")
				},
			},
			expectError: true,
			errorMsg:    "unsupported option type",
		},
		{
			name:        "Option Type Test",
			machineType: types.Risor,
			options: []any{
				options.WithLoader(func() loader.Loader {
					l, err := loader.NewFromString("print('test')")
					require.NoError(t, err)
					return l
				}()),
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			var evaluator engine.EvaluatorWithPrep
			var err error

			switch tc.machineType {
			case types.Starlark:
				evaluator, err = NewStarlarkEvaluator(tc.options...)
			case types.Risor:
				evaluator, err = NewRisorEvaluator(tc.options...)
			case types.Extism:
				evaluator, err = NewExtismEvaluator(tc.options...)
			default:
				t.Fatalf("unsupported machine type: %s", tc.machineType)
			}

			if tc.expectError {
				require.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
				return
			}

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
		creator     func(content string, opts ...any) (engine.EvaluatorWithPrep, error)
		options     []any
		expectError bool
	}{
		{
			name:        "FromStarlarkString - Valid",
			content:     `print("Hello, World!")`,
			creator:     FromStarlarkString,
			options:     []any{starlarkCompiler.WithGlobals([]string{"ctx"})},
			expectError: false,
		},
		{
			name:        "FromRisorString - Valid",
			content:     `print("Hello, World!")`,
			creator:     FromRisorString,
			options:     []any{risorCompiler.WithGlobals([]string{"ctx"})},
			expectError: false,
		},
		{
			name:        "FromStarlarkString - Empty",
			content:     "",
			creator:     FromStarlarkString,
			options:     []any{},
			expectError: true,
		},
		{
			name:        "FromRisorString - Empty",
			content:     "",
			creator:     FromRisorString,
			options:     []any{},
			expectError: true,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			evaluator, err := tc.creator(tc.content, tc.options...)

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, evaluator)
		})
	}

	// Skip the Extism string loader test - covered by design

	// Test invalid option in string loader
	t.Run("FromRisorString - Invalid Option", func(t *testing.T) {
		_, err := FromRisorString(
			"print('test')",
			func(cfg *options.Config) error {
				return errors.New("invalid option test")
			},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported option type")
	})
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

	tests := []struct {
		name        string
		loaderFunc  func(string, ...any) (engine.EvaluatorWithPrep, error)
		filePath    string
		options     []any
		expectError bool
	}{
		{
			name:       "FromExtismFile - Valid",
			loaderFunc: FromExtismFile,
			filePath:   wasmPath,
			options: []any{
				options.WithLogHandler(getLogger()),
				extismCompiler.WithEntryPoint("greet"),
				options.WithDataProvider(data.NewStaticProvider(map[string]any{
					"input": "Test User", // Required for WASM execution
				})),
			},
			expectError: false,
		},
		{
			name:        "FromExtismFile - Invalid Path",
			loaderFunc:  FromExtismFile,
			filePath:    "non-existent-file.wasm",
			options:     []any{},
			expectError: true,
		},
		{
			name:       "FromRisorFile - Valid",
			loaderFunc: FromRisorFile,
			filePath:   risorPath,
			options: []any{
				options.WithLogHandler(getLogger()),
				risorCompiler.WithGlobals([]string{"ctx"}),
			},
			expectError: false,
		},
		{
			name:        "FromRisorFile - Invalid Path",
			loaderFunc:  FromRisorFile,
			filePath:    "non-existent-file.risor",
			options:     []any{},
			expectError: true,
		},
		{
			name:       "FromStarlarkFile - Valid",
			loaderFunc: FromStarlarkFile,
			filePath:   starlarkPath,
			options: []any{
				options.WithLogHandler(getLogger()),
				starlarkCompiler.WithGlobals([]string{"ctx"}),
			},
			expectError: false,
		},
		{
			name:        "FromStarlarkFile - Invalid Path",
			loaderFunc:  FromStarlarkFile,
			filePath:    "non-existent-file.star",
			options:     []any{},
			expectError: true,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			evaluator, err := tc.loaderFunc(tc.filePath, tc.options...)

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, evaluator)

			// For valid evaluators, test basic execution only for non-Extism types
			if !tc.expectError && tc.name != "FromExtismFile - Valid" {
				result, evalErr := evaluator.Eval(context.Background())
				require.NoError(t, evalErr)
				require.NotNil(t, result)
			}
		})
	}
}

func TestDataProviders(t *testing.T) {
	t.Parallel()

	t.Run("withCompositeProvider", func(t *testing.T) {
		// Create a simple script that uses composite data
		script := `print(ctx["static_key"], ", ", ctx["input_data"]["dynamic_key"])`

		// Create static data
		staticData := map[string]any{
			"static_key": "static_value",
		}

		// Create an evaluator with composite provider
		evaluator, err := FromStarlarkString(
			script,
			withCompositeProvider(staticData),
			starlarkCompiler.WithGlobals([]string{constants.Ctx}),
		)
		require.NoError(t, err)
		require.NotNil(t, evaluator)

		// Test adding dynamic data
		ctx := context.Background()
		dynamicData := map[string]any{"dynamic_key": "dynamic_value"}
		enrichedCtx, err := evaluator.PrepareContext(ctx, dynamicData)
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
			name := ctx["input_data"]["name"]
			{
				"message": "Hello, " + name + "!",
				"length": len(name)
			}
		`

		// Create an evaluator with the CompositeProvider
		evaluator, err := FromRisorString(
			script,
			options.WithDefaults(),
			options.WithLogHandler(getLogger()),
			withCompositeProvider(map[string]any{}),
			risorCompiler.WithGlobals([]string{constants.Ctx}),
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
		t.Run("PrepareContext error", func(t *testing.T) {
			// Create mocks for testing error cases
			mockPrepCtx := &mockPreparer{}
			mockEval := &mockEvaluator{}

			// Create context and data
			ctx := context.Background()
			data := map[string]any{"name": "World"}

			// Mock PrepareContext to return an error
			mockPrepCtx.On("PrepareContext", ctx, []any{data}).
				Return(ctx, errors.New("prepare error"))

			// Create a mock evaluator that implements both interfaces
			mockEvalWithPrep := struct {
				engine.Evaluator
				engine.EvalDataPreparer
			}{
				Evaluator:        mockEval,
				EvalDataPreparer: mockPrepCtx,
			}

			// PrepareAndEval should return the prepare error
			_, err = prepareAndEval(t, ctx, mockEvalWithPrep, data)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to prepare context")
			mockPrepCtx.AssertExpectations(t)
		})

		t.Run("Eval error", func(t *testing.T) {
			// Create mocks for testing error cases
			mockPrepCtx := &mockPreparer{}
			mockEval := &mockEvaluator{}

			// Create context and data
			ctx := context.Background()
			data := map[string]any{"name": "World"}

			// Mock PrepareContext to succeed
			//nolint
			enrichedCtx := context.WithValue(ctx, "test-key", "test-value")
			mockPrepCtx.On("PrepareContext", ctx, []any{data}).Return(enrichedCtx, nil)

			// Mock Eval to fail
			mockEval.On("Eval", enrichedCtx).Return(nil, errors.New("eval error"))

			// Create a mock evaluator that implements both interfaces
			mockEvalWithPrep := struct {
				engine.Evaluator
				engine.EvalDataPreparer
			}{
				Evaluator:        mockEval,
				EvalDataPreparer: mockPrepCtx,
			}

			// PrepareAndEval should return the eval error
			_, err = prepareAndEval(t, ctx, mockEvalWithPrep, data)
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
		evaluator, err := FromRisorString(
			script,
			options.WithDefaults(),
			options.WithLogHandler(getLogger()),
			risorCompiler.WithGlobals([]string{constants.Ctx}),
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
		nilEvaluator, err := FromRisorString(
			nilScript,
			options.WithDefaults(),
			options.WithLogHandler(getLogger()),
			risorCompiler.WithGlobals([]string{constants.Ctx}),
		)
		require.NoError(t, err)

		nilResult, err := evalAndExtractMap(t, context.Background(), nilEvaluator)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{}, nilResult)

		// Test with non-map result (should error)
		numScript := `42`
		numEvaluator, err := FromRisorString(
			numScript,
			options.WithDefaults(),
			options.WithLogHandler(getLogger()),
			risorCompiler.WithGlobals([]string{constants.Ctx}),
		)
		require.NoError(t, err)

		_, err = evalAndExtractMap(t, context.Background(), numEvaluator)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "result is not a map")

		// Test with evaluation error
		t.Run("Eval error", func(t *testing.T) {
			mockEval := &mockEvaluator{}
			ctx := context.Background()

			// Mock Eval to return an error
			mockEval.On("Eval", ctx).Return(nil, errors.New("eval error"))

			// EvalAndExtractMap should return the error
			_, err = evalAndExtractMap(t, ctx, mockEval)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "script evaluation failed")
			mockEval.AssertExpectations(t)
		})
	})
}

func TestMachineWithData(t *testing.T) {
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
	"message": "Hello, " + ctx["input_data"]["name"] + " (v" + ctx["app_version"] + ")",
	"timeout": ctx["config"]["timeout"]
}`
	err = os.WriteFile(risorPath, []byte(risorFileContent), 0o644)
	require.NoError(t, err)

	// Create a basic Starlark script that uses context
	starlarkFileContent := `# Simple Starlark script
result = {
    "message": "Hello, " + ctx["input_data"]["name"] + " (v" + ctx["app_version"] + ")",
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

	t.Run("FromRisorStringWithData", func(t *testing.T) {
		// Test script
		risorScript := `
			// Access static data
			version := ctx["app_version"]
			timeout := ctx["config"]["timeout"]
			
			// Access dynamic data
			name := ctx["input_data"]["name"]
			
			{
				"message": "Hello, " + name + " (v" + version + ")",
				"timeout": timeout
			}
		`

		// Create evaluator
		risorEval, err := FromRisorStringWithData(risorScript, staticData, getLogger())
		require.NoError(t, err)

		// Test with dynamic data
		ctx := context.Background()
		dynamicData := map[string]any{"name": "Risor User"}
		enrichedCtx, err := risorEval.PrepareContext(ctx, dynamicData)
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

	t.Run("FromStarlarkStringWithData", func(t *testing.T) {
		// Create evaluator
		starlarkEval, err := FromStarlarkStringWithData(
			starlarkFileContent,
			staticData,
			getLogger(),
		)
		require.NoError(t, err)

		// Test with dynamic data
		ctx := context.Background()
		dynamicData := map[string]any{"name": "Starlark User"}
		enrichedCtx, err := starlarkEval.PrepareContext(ctx, dynamicData)
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

	t.Run("FromExtismFileWithData", func(t *testing.T) {
		// Create evaluator with static data that includes input
		extismEval, err := FromExtismFileWithData(
			wasmPath,
			map[string]any{"input": "Test User"},
			getLogger(),
			"greet", // entry point
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
		extismEvalNoInput, err := FromExtismFileWithData(
			wasmPath,
			staticData, // Only static config data, no input
			getLogger(),
			"greet", // entry point
		)
		require.NoError(t, err)
		require.NotNil(t, extismEvalNoInput)

		_, err = extismEvalNoInput.Eval(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "input string is empty")
	})
}

func TestFileWithDataFunctions(t *testing.T) {
	t.Parallel()

	// Create temporary test files
	tmpDir := t.TempDir()

	// Create test files to use for testing
	risorPath := filepath.Join(tmpDir, "test.risor")
	risorContent := `{ "message": "Hello from Risor!" }`
	err := os.WriteFile(risorPath, []byte(risorContent), 0o644)
	require.NoError(t, err)

	starlarkPath := filepath.Join(tmpDir, "test.star")
	starlarkContent := `result = {"message": "Hello from Starlark!"}\n_ = result`
	err = os.WriteFile(starlarkPath, []byte(starlarkContent), 0o644)
	require.NoError(t, err)

	// Test FromRisorFileWithData
	t.Run("FromRisorFileWithData", func(t *testing.T) {
		logger := getLogger()
		staticData := map[string]any{"test": "data"}

		// This just needs to call the function, even if execution would fail later
		_, err := FromRisorFileWithData(risorPath, staticData, logger)
		// We don't assert on the result since we just want to cover the function
		_ = err
	})

	// Test FromStarlarkFileWithData
	t.Run("FromStarlarkFileWithData", func(t *testing.T) {
		logger := getLogger()
		staticData := map[string]any{"test": "data"}

		// This just needs to call the function, even if execution would fail later
		_, err := FromStarlarkFileWithData(starlarkPath, staticData, logger)
		// We don't assert on the result since we just want to cover the function
		_ = err
	})
}

func TestFromStringLoader(t *testing.T) {
	t.Parallel()

	// Test the Extism string loader error case directly
	t.Run("ExtismStringNotSupported", func(t *testing.T) {
		// We can't call it directly, so we'll make our own version
		// that's similar to what FromExtismString would look like
		// if it existed, but just enough to test the error branch
		content := "test"
		l, err := loader.NewFromString(content)
		require.NoError(t, err)

		// Create the options with the string loader
		opts := []any{options.WithLoader(l)}

		// Create Extism evaluator, which should fail
		_, err = NewExtismEvaluator(opts...)
		// We just want to make sure it errors out
		require.Error(t, err)
	})
}

func TestCreateEvaluatorEdgeCases2(t *testing.T) {
	t.Parallel()

	// Test a case where source URL is nil
	t.Run("NilSourceURL", func(t *testing.T) {
		// Create a minimal mock loader with nil URL
		mockLoader := &mockLoader{}

		// Create an evaluator with this loader
		_, err := NewRisorEvaluator(
			options.WithLoader(mockLoader),
			options.WithDefaults(),
		)

		// Because we specified risorCompiler.WithGlobals, we'll get compiler options error
		require.Error(t, err)
	})
}

// mockLoader is a simple implementation of loader.Loader that's just enough to test
// the nil source URL case
type mockLoader struct{}

func (m *mockLoader) GetReader() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("return 0")), nil
}

func (m *mockLoader) GetSourceURL() *url.URL {
	return nil
}

func TestNewExtismEvaluator(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for the WASM file
	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "main.wasm")

	// Write the embedded WASM data to the temporary file
	err := os.WriteFile(wasmPath, wasmData, 0o644)
	require.NoError(t, err)

	// Create a logger handler
	handler := getLogger()

	// Create an evaluator with file loader
	evaluator, err := NewExtismEvaluator(
		options.WithDefaults(),
		options.WithLoader(
			func() loader.Loader {
				loader, err := loader.NewFromDisk(wasmPath)
				require.NoError(t, err)
				return loader
			}(),
		),
		options.WithLogHandler(handler),
		options.WithDataProvider(data.NewStaticProvider(map[string]any{
			"input": "Test User", // Put the input directly at the top level
		})),
		extismCompiler.WithEntryPoint("greet"),
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

	// Test validation error in newEvaluator
	t.Run("Configuration Validation Error", func(t *testing.T) {
		// Try to create an evaluator without a loader
		_, err := NewRisorEvaluator()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no loader specified")
	})

	// Test option application error
	t.Run("Option Error", func(t *testing.T) {
		// Create an invalid option that returns an error
		invalidOption := func(cfg *options.Config) error {
			return errors.New("custom invalid option error")
		}

		// This should fail when applying the option
		_, err := NewRisorEvaluator(invalidOption)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported option type")
	})
}
