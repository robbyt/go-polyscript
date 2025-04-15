package platform_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/mocks"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockDataPreparer is a mock implementation of EvalDataPreparer
type mockDataPreparer struct {
	mock.Mock
}

func (m *mockDataPreparer) AddDataToContext(
	ctx context.Context,
	data ...map[string]any,
) (context.Context, error) {
	args := m.Called(ctx, data)
	return args.Get(0).(context.Context), args.Error(1)
}

// mockEvaluatorWithPreparer creates an evaluator implementation that satisfies both interfaces
type mockEvaluatorWithPreparer struct {
	mock.Mock
}

func (m *mockEvaluatorWithPreparer) Eval(
	ctx context.Context,
) (platform.EvaluatorResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(platform.EvaluatorResponse), args.Error(1)
}

func (m *mockEvaluatorWithPreparer) AddDataToContext(
	ctx context.Context,
	data ...map[string]any,
) (context.Context, error) {
	args := m.Called(ctx, data)
	return args.Get(0).(context.Context), args.Error(1)
}

func TestEvaluatorInterface(t *testing.T) {
	t.Parallel()
	// Create a mock evaluator response
	mockResponse := new(mocks.EvaluatorResponse)
	mockResponse.On("Interface").Return("test result")
	mockResponse.On("GetScriptExeID").Return("test-script-id")
	mockResponse.On("GetExecTime").Return("10µs")
	mockResponse.On("Type").Return(data.STRING)
	mockResponse.On("Inspect").Return("test result")

	// use a custom type for the context key lookup, to avoid lint warnings
	type contextKey string
	testKey := contextKey("test-key")

	// Create a context with a test key
	ctx := context.WithValue(context.Background(), testKey, "test-value")

	// Create a mock evaluator with success case
	evaluator := new(mocks.Evaluator)
	evaluator.On("Eval", mock.MatchedBy(func(c context.Context) bool {
		// Verify that context is passed correctly
		_, hasKey := c.Value(testKey).(string)
		return hasKey
	})).Return(mockResponse, nil)

	// Test the Eval method with the context
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
	errorEvaluator := new(mocks.Evaluator)
	errorEvaluator.On("Eval", mock.Anything).
		Return((*mocks.EvaluatorResponse)(nil), errors.New("evaluation error"))

	response, err = errorEvaluator.Eval(context.Background())
	assert.Error(t, err, "Eval should return an error")
	assert.Nil(t, response, "Response should be nil when there's an error")
	assert.Contains(t, err.Error(), "evaluation error", "Error message should be preserved")
}

func TestEvalDataPreparerInterface(t *testing.T) {
	t.Parallel()
	// Create a logger for testing
	handler := slog.NewTextHandler(os.Stdout, nil)

	// Create an evaluator with AddDataToContext capability
	// The key name may be different in the new implementation
	scriptData := map[string]any{"greeting": "Hello, World!"}
	evaluator, err := polyscript.FromRisorStringWithData(`
method := ctx["request"]["Method"] 
greeting := ctx["greeting"]  // With new implementation, keys are at top level
method + " " + greeting
`,
		scriptData,
		handler,
	)
	require.NoError(t, err)
	require.NotNil(t, evaluator)

	// Create context and test data
	ctx := context.Background()
	req, err := http.NewRequest("GET", "http://localhost/test", nil)
	require.NoError(t, err)

	// Use AddDataToContext to enrich the context
	enrichedCtx, err := evaluator.AddDataToContext(ctx, map[string]any{"request": req})
	require.NoError(t, err)
	require.NotNil(t, enrichedCtx)

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
	t.Parallel()
	dataPreparer := &mockDataPreparer{}

	// Test with various data types
	ctx := context.Background()
	data1 := "string data"
	data2 := map[string]any{"key": "value"}
	data3 := 123

	// Create enriched context with the test data
	enrichedCtx := ctx
	type dataKey string
	for i, item := range []any{data1, data2, data3} {
		key := dataKey(fmt.Sprintf("data-%d", i))
		enrichedCtx = context.WithValue(enrichedCtx, key, item)
		require.NotNil(t, enrichedCtx)
	}

	// Set up the mock behavior
	dataPreparer.On("AddDataToContext", ctx, mock.Anything).Return(enrichedCtx, nil)

	// Call AddDataToContext
	resultCtx, err := dataPreparer.AddDataToContext(
		ctx,
		map[string]any{"data1": data1},
		map[string]any{"data2": data2},
		map[string]any{"data3": data3},
	)
	require.NoError(t, err, "AddDataToContext should not return an error")
	require.NotNil(t, resultCtx, "Enriched context should not be nil")

	// Verify data was stored correctly
	for i, item := range []any{data1, data2, data3} {
		key := dataKey(fmt.Sprintf("data-%d", i))
		storedItem := resultCtx.Value(key)
		require.NotNil(t, storedItem, "Stored item should not be nil")
		assert.Equal(t, item, storedItem, "Stored item should match original data")
	}

	// Test error case
	errorPreparer := &mockDataPreparer{}
	errorPreparer.On("AddDataToContext", ctx, mock.Anything).
		Return(ctx, errors.New("preparation error"))

	ogCtx, err := errorPreparer.AddDataToContext(ctx, map[string]any{"test": "value"})
	assert.Error(t, err, "Should return an error")
	assert.ErrorContains(t, err, "preparation error", "Error message should be preserved")
	assert.Equal(t, ctx, ogCtx, "Original context should be returned on error")
}

func TestEvaluatorWithPrepInterface(t *testing.T) {
	t.Parallel()
	// Create a mock evaluator response
	mockResponse := new(mocks.EvaluatorResponse)
	mockResponse.On("Interface").Return("combined result")
	mockResponse.On("GetScriptExeID").Return("test-script-id")
	mockResponse.On("GetExecTime").Return("10µs")
	mockResponse.On("Type").Return(data.STRING)
	mockResponse.On("Inspect").Return("combined result")

	// use a custom type for the context key lookup, to avoid lint warnings
	type prepKey string
	prepDataKey := prepKey("prepared-data")

	// Create a mock combined implementation
	combinedEvaluator := &mockEvaluatorWithPreparer{}

	// Define context and test data
	ctx := context.Background()
	enrichedCtx := context.WithValue(ctx, prepDataKey, "test-value")

	// Set up mock behaviors
	combinedEvaluator.On("AddDataToContext", ctx, mock.Anything).Return(enrichedCtx, nil)
	combinedEvaluator.On("Eval", mock.MatchedBy(func(c context.Context) bool {
		val, ok := c.Value(prepDataKey).(string)
		return ok && val == "test-value"
	})).Return(mockResponse, nil)

	// Test the full workflow: prepare context then evaluate
	resultCtx, err := combinedEvaluator.AddDataToContext(ctx, map[string]any{"test": "data"})
	require.NoError(t, err, "AddDataToContext should not return an error")
	require.NotNil(t, resultCtx, "Enriched context should not be nil")

	// Then evaluate with the enriched context
	response, err := combinedEvaluator.Eval(resultCtx)
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
	prepErrorEvaluator := &mockEvaluatorWithPreparer{}
	prepErrorEvaluator.On("AddDataToContext", ctx, mock.Anything).
		Return(ctx, errors.New("preparation error"))

	_, err = prepErrorEvaluator.AddDataToContext(ctx, map[string]any{"test": "data"})
	assert.Error(t, err, "Should return an error when preparation fails")

	// Test error in evaluation
	evalErrorEvaluator := &mockEvaluatorWithPreparer{}
	evalErrorEvaluator.On("AddDataToContext", ctx, mock.Anything).Return(enrichedCtx, nil)
	evalErrorEvaluator.On("Eval", mock.Anything).
		Return((*mocks.EvaluatorResponse)(nil), errors.New("evaluation error"))

	evalCtx, prepErr := evalErrorEvaluator.AddDataToContext(ctx, map[string]any{"test": "data"})
	require.NoError(t, prepErr, "AddDataToContext should not return an error")
	_, err = evalErrorEvaluator.Eval(evalCtx)
	assert.Error(t, err, "Should return an error when evaluation fails")
}

func TestEvaluatorWithPrepErrors(t *testing.T) {
	t.Parallel()
	// Create a logger for testing
	handler := slog.NewTextHandler(os.Stdout, nil)

	// Test with StaticProvider (only testing specific error cases)
	staticData := map[string]any{"static": "data"}
	evaluator, err := polyscript.FromRisorStringWithData(`ctx["static"]`,
		staticData,
		handler,
	)
	require.NoError(t, err)

	// The context should still be usable
	ctx := context.Background()
	enrichedCtx, err := evaluator.AddDataToContext(
		ctx,
		map[string]any{"value": 123},
	) // Properly wrapped in map

	// This should now succeed as integers are properly wrapped in a map
	assert.NoError(t, err, "AddDataToContext should succeed with properly wrapped integers")
	assert.NotNil(t, enrichedCtx, "Should return a context regardless")
}
