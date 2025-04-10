package engine_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/engine/options"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/mocks"
	risorCompiler "github.com/robbyt/go-polyscript/machines/risor/compiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEvaluatorImplementation is a minimal struct implementing engine.Evaluator for testing
type mockEvaluatorImplementation struct {
	evalFunc func(ctx context.Context) (engine.EvaluatorResponse, error)
}

func (m *mockEvaluatorImplementation) Eval(ctx context.Context) (engine.EvaluatorResponse, error) {
	return m.evalFunc(ctx)
}

// mockEvalDataPreparerImplementation is a minimal struct implementing engine.EvalDataPreparer for testing
type mockEvalDataPreparerImplementation struct {
	prepareFunc func(ctx context.Context, data ...any) (context.Context, error)
}

func (m *mockEvalDataPreparerImplementation) PrepareContext(
	ctx context.Context,
	data ...any,
) (context.Context, error) {
	return m.prepareFunc(ctx, data...)
}

// mockEvaluatorWithPrepImplementation is a minimal struct implementing engine.EvaluatorWithPrep for testing
type mockEvaluatorWithPrepImplementation struct {
	evalFunc    func(ctx context.Context) (engine.EvaluatorResponse, error)
	prepareFunc func(ctx context.Context, data ...any) (context.Context, error)
}

func (m *mockEvaluatorWithPrepImplementation) Eval(
	ctx context.Context,
) (engine.EvaluatorResponse, error) {
	return m.evalFunc(ctx)
}

func (m *mockEvaluatorWithPrepImplementation) PrepareContext(
	ctx context.Context,
	data ...any,
) (context.Context, error) {
	return m.prepareFunc(ctx, data...)
}

func TestEvaluatorInterface(t *testing.T) {
	// Create a mock evaluator response
	mockResponse := new(mocks.EvaluatorResponse)
	mockResponse.On("Interface").Return("test result")
	mockResponse.On("GetScriptExeID").Return("test-script-id")
	mockResponse.On("GetExecTime").Return("10µs")
	mockResponse.On("Type").Return(data.STRING)
	mockResponse.On("Inspect").Return("test result")

	// Define a type for the context key to avoid collision
	type contextKey string
	testKey := contextKey("test-key")

	// Create a mock evaluator implementation
	evaluator := &mockEvaluatorImplementation{
		evalFunc: func(ctx context.Context) (engine.EvaluatorResponse, error) {
			// Verify that context is passed correctly
			_, hasKey := ctx.Value(testKey).(string)
			if !hasKey {
				return nil, errors.New("context key missing")
			}
			return mockResponse, nil
		},
	}

	// Test the Eval method with a context containing a test key
	ctx := context.WithValue(context.Background(), testKey, "test-value")
	response, err := evaluator.Eval(ctx)

	require.NoError(t, err, "Eval should not return an error")
	require.NotNil(t, response, "Response should not be nil")

	// Verify response methods
	assert.Equal(t, "test result", response.Interface(), "Interface() should return expected value")
	assert.Equal(
		t,
		"test-script-id",
		response.GetScriptExeID(),
		"GetScriptExeID() should return expected value",
	)
	assert.Equal(t, "10µs", response.GetExecTime(), "GetExecTime() should return expected value")
	assert.Equal(t, data.STRING, response.Type(), "Type() should return expected value")
	assert.Equal(t, "test result", response.Inspect(), "Inspect() should return expected value")

	// Test error case
	evaluator.evalFunc = func(ctx context.Context) (engine.EvaluatorResponse, error) {
		return nil, errors.New("evaluation error")
	}

	response, err = evaluator.Eval(context.Background())
	assert.Error(t, err, "Eval should return an error")
	assert.Nil(t, response, "Response should be nil when there's an error")
	assert.Contains(t, err.Error(), "evaluation error", "Error message should be preserved")
}

func TestEvalDataPreparerInterface(t *testing.T) {
	// Create a logger for testing
	handler := slog.NewTextHandler(os.Stdout, nil)

	// Create a ContextProvider for this test
	provider := data.NewContextProvider(constants.EvalData)

	// Create an evaluator with PrepareContext capability
	evaluator, err := polyscript.FromRisorString(`
method := ctx["request"]["Method"] 
greeting := ctx["input_data"]["greeting"]
method + " " + greeting
`,
		options.WithLogHandler(handler),
		options.WithDataProvider(provider),
		risorCompiler.WithGlobals([]string{constants.Ctx}),
	)
	require.NoError(t, err)
	require.NotNil(t, evaluator)

	// Create context and test data
	ctx := context.Background()
	req, err := http.NewRequest("GET", "http://localhost/test", nil)
	require.NoError(t, err)
	scriptData := map[string]any{"greeting": "Hello, World!"}

	// Use PrepareContext to enrich the context
	enrichedCtx, err := evaluator.PrepareContext(ctx, req, scriptData)
	require.NoError(t, err)
	require.NotNil(t, enrichedCtx)

	// Verify data was stored in context
	storedData, ok := enrichedCtx.Value(constants.EvalData).(map[string]any)
	require.True(t, ok, "Data should be stored in context")
	require.NotNil(t, storedData, "Stored data should not be nil")

	// Verify request data
	requestData, ok := storedData[constants.Request].(map[string]any)
	require.True(t, ok, "Request data should be available")
	assert.Equal(t, "GET", requestData["Method"], "Request method should be stored")

	// Verify script data
	scriptDataStored, ok := storedData[constants.InputData].(map[string]any)
	require.True(t, ok, "input_data should be available")
	assert.Equal(t, "Hello, World!", scriptDataStored["greeting"], "Greeting should be stored")

	// Test evaluation with the enriched context
	result, err := evaluator.Eval(enrichedCtx)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify result
	assert.Equal(
		t,
		"GET Hello, World!",
		fmt.Sprintf("%v", result.Interface()),
		"Script result should match expected",
	)
}

func TestEvalDataPreparerInterfaceDirectImplementation(t *testing.T) {
	// Define a type for the context key to avoid collision
	type dataKey string

	// Create a mock data preparer implementation
	dataPreparer := &mockEvalDataPreparerImplementation{
		prepareFunc: func(ctx context.Context, data ...any) (context.Context, error) {
			// Simple implementation to store data in context
			enrichedCtx := ctx
			for i, item := range data {
				key := dataKey(fmt.Sprintf("data-%d", i))
				enrichedCtx = context.WithValue(enrichedCtx, key, item)
			}
			return enrichedCtx, nil
		},
	}

	// Test with various data types
	ctx := context.Background()
	data1 := "string data"
	data2 := map[string]any{"key": "value"}
	data3 := 123

	enrichedCtx, err := dataPreparer.PrepareContext(ctx, data1, data2, data3)
	require.NoError(t, err, "PrepareContext should not return an error")
	require.NotNil(t, enrichedCtx, "Enriched context should not be nil")

	// Verify data was stored correctly
	assert.Equal(
		t,
		data1,
		enrichedCtx.Value(dataKey("data-0")),
		"First data item should be stored correctly",
	)
	assert.Equal(
		t,
		data2,
		enrichedCtx.Value(dataKey("data-1")),
		"Second data item should be stored correctly",
	)
	assert.Equal(
		t,
		data3,
		enrichedCtx.Value(dataKey("data-2")),
		"Third data item should be stored correctly",
	)

	// Test error case
	dataPreparer.prepareFunc = func(ctx context.Context, data ...any) (context.Context, error) {
		return ctx, errors.New("preparation error")
	}

	_, err = dataPreparer.PrepareContext(ctx, "test")
	assert.Error(t, err, "Should return an error")
	assert.Contains(t, err.Error(), "preparation error", "Error message should be preserved")
}

func TestEvaluatorWithPrepInterface(t *testing.T) {
	// Create a mock evaluator response
	mockResponse := new(mocks.EvaluatorResponse)
	mockResponse.On("Interface").Return("combined result")
	mockResponse.On("GetScriptExeID").Return("test-script-id")
	mockResponse.On("GetExecTime").Return("10µs")
	mockResponse.On("Type").Return(data.STRING)
	mockResponse.On("Inspect").Return("combined result")

	// Define a type for the context key to avoid collision
	type prepKey string
	prepDataKey := prepKey("prepared-data")

	// Create a mock combined implementation
	combinedEvaluator := &mockEvaluatorWithPrepImplementation{
		evalFunc: func(ctx context.Context) (engine.EvaluatorResponse, error) {
			// Check if context has prepared data
			value, ok := ctx.Value(prepDataKey).(string)
			if !ok || value != "test-value" {
				return nil, errors.New("context not properly prepared")
			}
			return mockResponse, nil
		},
		prepareFunc: func(ctx context.Context, data ...any) (context.Context, error) {
			// Store the prepared data in context
			return context.WithValue(ctx, prepDataKey, "test-value"), nil
		},
	}

	// Test the full workflow: prepare context then evaluate
	ctx := context.Background()

	// First prepare the context
	enrichedCtx, err := combinedEvaluator.PrepareContext(ctx, "test data")
	require.NoError(t, err, "PrepareContext should not return an error")
	require.NotNil(t, enrichedCtx, "Enriched context should not be nil")

	// Then evaluate with the enriched context
	response, err := combinedEvaluator.Eval(enrichedCtx)
	require.NoError(t, err, "Eval should not return an error when context is prepared")
	require.NotNil(t, response, "Response should not be nil")

	// Verify the response
	assert.Equal(
		t,
		"combined result",
		response.Interface(),
		"Interface() should return expected value",
	)

	// Test error in preparation
	combinedEvaluator.prepareFunc = func(ctx context.Context, data ...any) (context.Context, error) {
		return ctx, errors.New("preparation error")
	}

	_, err = combinedEvaluator.PrepareContext(ctx, "test data")
	assert.Error(t, err, "Should return an error when preparation fails")

	// Test error in evaluation
	combinedEvaluator.prepareFunc = func(ctx context.Context, data ...any) (context.Context, error) {
		return context.WithValue(ctx, prepDataKey, "test-value"), nil
	}
	combinedEvaluator.evalFunc = func(ctx context.Context) (engine.EvaluatorResponse, error) {
		return nil, errors.New("evaluation error")
	}

	enrichedCtx, prepErr := combinedEvaluator.PrepareContext(ctx, "test data")
	require.NoError(t, prepErr, "PrepareContext should not return an error")
	_, err = combinedEvaluator.Eval(enrichedCtx)
	assert.Error(t, err, "Should return an error when evaluation fails")
}

func TestEvaluatorWithPrepErrors(t *testing.T) {
	// Create a logger for testing
	handler := slog.NewTextHandler(os.Stdout, nil)

	// Test with StaticProvider (which doesn't support adding data)
	staticProvider := data.NewStaticProvider(map[string]any{"static": "data"})
	evaluator, err := polyscript.FromRisorString(`ctx["static"]`,
		options.WithLogHandler(handler),
		options.WithDataProvider(staticProvider),
		risorCompiler.WithGlobals([]string{constants.Ctx}),
	)
	require.NoError(t, err)

	// Try to prepare context with StaticProvider
	ctx := context.Background()
	_, err = evaluator.PrepareContext(ctx, map[string]any{"greeting": "Hello"})

	// Should return error about StaticProvider not supporting runtime data changes
	assert.Error(t, err, "Should return error for static provider")
	assert.Contains(
		t,
		err.Error(),
		"StaticProvider doesn't support adding data",
		"Error should mention static provider limitation",
	)

	// Test with evaluator that has a ContextProvider
	contextProvider := data.NewContextProvider(constants.EvalData)
	evaluator, err = polyscript.FromRisorString(`ctx["request"]["ID"] || "no id"`,
		options.WithLogHandler(handler),
		options.WithDataProvider(contextProvider),
		risorCompiler.WithGlobals([]string{constants.Ctx}),
	)
	require.NoError(t, err)

	// Try to prepare context with unsupported data
	enrichedCtx, err := evaluator.PrepareContext(ctx, 123) // Integer not supported directly

	// Should return error about unsupported data type, but still return the context
	assert.Error(t, err, "Should return error for unsupported data type")
	assert.Contains(
		t,
		err.Error(),
		"unsupported data type",
		"Error should mention unsupported data type",
	)

	// The context should still be usable
	assert.NotNil(t, enrichedCtx, "Should still return a context even with error")
}
