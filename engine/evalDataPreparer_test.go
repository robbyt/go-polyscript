package engine_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/risor"
	"github.com/robbyt/go-polyscript/options"
)

func TestEvalDataPreparerInterface(t *testing.T) {
	// Create a logger for testing
	handler := slog.NewTextHandler(os.Stdout, nil)

	// Create a ContextProvider for this test
	provider := data.NewContextProvider(constants.EvalData)

	// Create an evaluator with PrepareContext capability
	evaluator, err := polyscript.FromRisorString(`
method := ctx["request"]["Method"] 
greeting := ctx["script_data"]["greeting"]
method + " " + greeting
`,
		options.WithLogger(handler),
		options.WithDataProvider(provider),
		risor.WithGlobals([]string{constants.Ctx}),
	)
	require.NoError(t, err)
	require.NotNil(t, evaluator)

	// Create context and test data
	ctx := context.Background()
	req, err := http.NewRequest("GET", "https://example.com", nil)
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
	scriptDataStored, ok := storedData[constants.ScriptData].(map[string]any)
	require.True(t, ok, "Script data should be available")
	assert.Equal(t, "Hello, World!", scriptDataStored["greeting"], "Greeting should be stored")

	// Test evaluation with the enriched context
	result, err := evaluator.Eval(enrichedCtx)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify result
	assert.Equal(t, "GET Hello, World!", fmt.Sprintf("%v", result.Interface()), "Script result should match expected")
}

func TestEvaluatorWithPrepErrors(t *testing.T) {
	// Create a logger for testing
	handler := slog.NewTextHandler(os.Stdout, nil)

	// Test with StaticProvider (which doesn't support adding data)
	staticProvider := data.NewStaticProvider(map[string]any{"static": "data"})
	evaluator, err := polyscript.FromRisorString(`ctx["static"]`,
		options.WithLogger(handler),
		options.WithDataProvider(staticProvider),
		risor.WithGlobals([]string{constants.Ctx}),
	)
	require.NoError(t, err)

	// Try to prepare context with StaticProvider
	ctx := context.Background()
	_, err = evaluator.PrepareContext(ctx, map[string]any{"greeting": "Hello"})

	// Should return error about StaticProvider not supporting runtime data changes
	assert.Error(t, err, "Should return error for static provider")
	assert.Contains(t, err.Error(), "StaticProvider doesn't support adding data", "Error should mention static provider limitation")

	// Test with evaluator that has a ContextProvider
	contextProvider := data.NewContextProvider(constants.EvalData)
	evaluator, err = polyscript.FromRisorString(`ctx["request"]["ID"] || "no id"`,
		options.WithLogger(handler),
		options.WithDataProvider(contextProvider),
		risor.WithGlobals([]string{constants.Ctx}),
	)
	require.NoError(t, err)

	// Try to prepare context with unsupported data
	enrichedCtx, err := evaluator.PrepareContext(ctx, 123) // Integer not supported directly

	// Should return error about unsupported data type, but still return the context
	assert.Error(t, err, "Should return error for unsupported data type")
	assert.Contains(t, err.Error(), "unsupported data type", "Error should mention unsupported data type")

	// The context should still be usable
	assert.NotNil(t, enrichedCtx, "Should still return a context even with error")
}
