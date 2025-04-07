package risor

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/risor-io/risor/compiler"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/machines/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock the data.Provider interface
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) GetData(ctx context.Context) (map[string]any, error) {
	args := m.Called(ctx)
	if data, ok := args.Get(0).(map[string]any); ok {
		return data, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProvider) AddDataToContext(ctx context.Context, data ...any) (context.Context, error) {
	args := m.Called(ctx, data)
	if ctx, ok := args.Get(0).(context.Context); ok {
		return ctx, args.Error(1)
	}
	return ctx, args.Error(1)
}

// MockLoader creates a simple loader for testing
type MockLoader struct {
	Content string
}

func (m *MockLoader) GetReader() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(m.Content)), nil
}

func (m *MockLoader) GetSourceURL() string {
	return "mock://source"
}

// MockContent implements ExecutableContent interface
type MockContent struct {
	Source  string
	Content any
}

func (m *MockContent) GetSource() string {
	return m.Source
}

func (m *MockContent) GetByteCode() any {
	return m.Content
}

func (m *MockContent) GetMachineType() types.Type {
	return types.Risor
}

// TestValidScript tests evaluating valid Risor scripts
func TestValidScript(t *testing.T) {
	t.Parallel()
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(handler))

	// Define the test script
	scriptContent := `
func handle(request) {
	if request == nil {
		return error("request is nil")
	}
	if request["Method"] == "POST" {
		return "post"
	}
	if request["URL_Path"] == "/hello" {
		return true
	}
	return false
}
print(ctx)
handle(ctx["request"])
`
	ld, err := loader.NewFromString(scriptContent)
	require.NoError(t, err)

	opt := &RisorOptions{Globals: []string{constants.Ctx}}

	// Create a context provider to use with our test context
	ctxProvider := data.NewContextProvider(constants.EvalData)

	exe, err := createTestExecutable(handler, ld, opt, ctxProvider)
	require.NoError(t, err)

	evaluator := NewBytecodeEvaluator(handler, exe)
	require.NotNil(t, evaluator)

	t.Run("get request", func(t *testing.T) {
		// Create the HttpRequest data object
		req := httptest.NewRequest("GET", "/hello", nil)
		rMap, err := helpers.RequestToMap(req)
		require.NoError(t, err)
		require.NotNil(t, rMap)
		require.Equal(t, "/hello", rMap["URL_Path"])

		evalData := map[string]any{
			constants.Request: rMap,
		}

		ctx := context.WithValue(context.Background(), constants.EvalData, evalData)

		// Evaluate the script with the provided HttpRequest
		response, err := evaluator.Eval(ctx)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Assert the response
		require.Equal(t, data.Types("bool"), response.Type())
		require.Equal(t, "true", response.Inspect())

		// Check the value
		boolValue, ok := response.Interface().(bool)
		require.True(t, ok)
		require.True(t, boolValue)
	})

	t.Run("post request", func(t *testing.T) {
		// Create the HttpRequest data object
		req := httptest.NewRequest("POST", "/hello", nil)
		rMap, err := helpers.RequestToMap(req)
		require.NoError(t, err)
		require.NotNil(t, rMap)
		require.Equal(t, "/hello", rMap["URL_Path"])

		evalData := map[string]any{
			constants.Request: rMap,
		}

		ctx := context.WithValue(context.Background(), constants.EvalData, evalData)

		// Evaluate the script with the provided HttpRequest
		response, err := evaluator.Eval(ctx)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Assert the response
		require.Equal(t, data.Types("string"), response.Type())
		require.Equal(t, "\"post\"", response.Inspect())

		// Check the value
		strValue, ok := response.Interface().(string)
		require.True(t, ok)
		require.Equal(t, "post", strValue)
	})
}

// TestString tests the String method
func TestString(t *testing.T) {
	t.Parallel()
	evaluator := &BytecodeEvaluator{}
	require.Equal(t, "risor.BytecodeEvaluator", evaluator.String())
}

// TestPrepareContext tests the PrepareContext method
func TestPrepareContext(t *testing.T) {
	t.Parallel()
	handler := slog.NewTextHandler(os.Stderr, nil)

	t.Run("with provider", func(t *testing.T) {
		// Setup the mock provider
		mockProvider := &MockProvider{}
		enrichedCtx := context.WithValue(context.Background(), constants.EvalData, "enriched")
		mockProvider.On("AddDataToContext", mock.Anything, mock.Anything).Return(enrichedCtx, nil)

		// Create an executable unit
		exe := &script.ExecutableUnit{DataProvider: mockProvider}

		// Create the evaluator
		evaluator := &BytecodeEvaluator{
			ctxKey:     constants.Ctx,
			execUnit:   exe,
			logHandler: handler,
			logger:     slog.New(handler),
		}

		// Call PrepareContext
		ctx := context.Background()
		data := map[string]any{"test": "data"}
		result, err := evaluator.PrepareContext(ctx, data)

		// Verify results
		require.NoError(t, err)
		require.Equal(t, enrichedCtx, result)
		mockProvider.AssertExpectations(t)
	})

	t.Run("with provider error", func(t *testing.T) {
		// Setup the mock provider
		mockProvider := &MockProvider{}
		expectedErr := fmt.Errorf("provider error")
		mockProvider.On("AddDataToContext", mock.Anything, mock.Anything).Return(nil, expectedErr)

		// Create an executable unit
		exe := &script.ExecutableUnit{DataProvider: mockProvider}

		// Create the evaluator
		evaluator := &BytecodeEvaluator{
			ctxKey:     constants.Ctx,
			execUnit:   exe,
			logHandler: handler,
			logger:     slog.New(handler),
		}

		// Call PrepareContext
		ctx := context.Background()
		data := map[string]any{"test": "data"}
		_, err := evaluator.PrepareContext(ctx, data)

		// Verify error is returned
		require.Error(t, err)
		require.ErrorIs(t, err, expectedErr)
		mockProvider.AssertExpectations(t)
	})

	t.Run("nil provider", func(t *testing.T) {
		// Create an executable unit without a provider
		exe := &script.ExecutableUnit{DataProvider: nil}

		// Create the evaluator
		evaluator := &BytecodeEvaluator{
			ctxKey:     constants.Ctx,
			execUnit:   exe,
			logHandler: handler,
			logger:     slog.New(handler),
		}

		// Call PrepareContext
		ctx := context.Background()
		data := map[string]any{"test": "data"}
		_, err := evaluator.PrepareContext(ctx, data)

		// Verify error is returned
		require.Error(t, err)
		require.Contains(t, err.Error(), "no data provider available")
	})

	t.Run("nil executable unit", func(t *testing.T) {
		// Create the evaluator without an executable unit
		evaluator := &BytecodeEvaluator{
			ctxKey:     constants.Ctx,
			execUnit:   nil,
			logHandler: handler,
			logger:     slog.New(handler),
		}

		// Call PrepareContext
		ctx := context.Background()
		data := map[string]any{"test": "data"}
		_, err := evaluator.PrepareContext(ctx, data)

		// Verify error is returned
		require.Error(t, err)
		require.Contains(t, err.Error(), "no data provider available")
	})
}

// TestEval tests edge cases for the Eval method
func TestEval(t *testing.T) {
	t.Parallel()
	handler := slog.NewTextHandler(os.Stderr, nil)

	t.Run("nil executable unit", func(t *testing.T) {
		evaluator := &BytecodeEvaluator{
			ctxKey:     constants.Ctx,
			execUnit:   nil,
			logHandler: handler,
			logger:     slog.New(handler),
		}

		ctx := context.Background()
		result, err := evaluator.Eval(ctx)

		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "executable unit is nil")
	})

	t.Run("nil bytecode", func(t *testing.T) {
		// Create an executable unit with nil bytecode
		exe := &script.ExecutableUnit{
			ID: "test-id",
			Content: &MockContent{
				Content: nil,
			},
		}

		evaluator := &BytecodeEvaluator{
			ctxKey:     constants.Ctx,
			execUnit:   exe,
			logHandler: handler,
			logger:     slog.New(handler),
		}

		ctx := context.Background()
		result, err := evaluator.Eval(ctx)

		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "bytecode is nil")
	})

	t.Run("empty execution id", func(t *testing.T) {
		// Create an executable unit with empty ID
		exe := &script.ExecutableUnit{
			ID: "",
			Content: &MockContent{
				Content: &compiler.Code{},
			},
		}

		evaluator := &BytecodeEvaluator{
			ctxKey:     constants.Ctx,
			execUnit:   exe,
			logHandler: handler,
			logger:     slog.New(handler),
		}

		ctx := context.Background()
		result, err := evaluator.Eval(ctx)

		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "exeID is empty")
	})

	t.Run("wrong bytecode type", func(t *testing.T) {
		// Create an executable unit with wrong bytecode type
		exe := &script.ExecutableUnit{
			ID: "test-id",
			Content: &MockContent{
				Content: "not a risor bytecode",
			},
		}

		evaluator := &BytecodeEvaluator{
			ctxKey:     constants.Ctx,
			execUnit:   exe,
			logHandler: handler,
			logger:     slog.New(handler),
		}

		ctx := context.Background()
		result, err := evaluator.Eval(ctx)

		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "unable to type assert bytecode")
	})
}

// TestLoadInputData tests the loadInputData method
func TestLoadInputData(t *testing.T) {
	t.Parallel()
	handler := slog.NewTextHandler(os.Stderr, nil)

	t.Run("nil provider", func(t *testing.T) {
		evaluator := &BytecodeEvaluator{
			ctxKey:     constants.Ctx,
			execUnit:   nil,
			logHandler: handler,
			logger:     slog.New(handler),
		}

		ctx := context.Background()
		data, err := evaluator.loadInputData(ctx)

		require.NoError(t, err)
		require.NotNil(t, data)
		require.Empty(t, data)
	})

	t.Run("with provider error", func(t *testing.T) {
		// Setup the mock provider
		mockProvider := &MockProvider{}
		expectedErr := fmt.Errorf("provider error")
		mockProvider.On("GetData", mock.Anything).Return(nil, expectedErr)

		// Create an executable unit
		exe := &script.ExecutableUnit{
			DataProvider: mockProvider,
		}

		evaluator := &BytecodeEvaluator{
			ctxKey:     constants.Ctx,
			execUnit:   exe,
			logHandler: handler,
			logger:     slog.New(handler),
		}

		ctx := context.Background()
		data, err := evaluator.loadInputData(ctx)

		require.Error(t, err)
		require.Equal(t, expectedErr, err)
		require.Nil(t, data)
		mockProvider.AssertExpectations(t)
	})

	t.Run("with empty data", func(t *testing.T) {
		// Setup the mock provider
		mockProvider := &MockProvider{}
		emptyData := map[string]any{}
		mockProvider.On("GetData", mock.Anything).Return(emptyData, nil)

		// Create an executable unit
		exe := &script.ExecutableUnit{
			DataProvider: mockProvider,
		}

		evaluator := &BytecodeEvaluator{
			ctxKey:     constants.Ctx,
			execUnit:   exe,
			logHandler: handler,
			logger:     slog.New(handler),
		}

		ctx := context.Background()
		data, err := evaluator.loadInputData(ctx)

		require.NoError(t, err)
		require.Empty(t, data)
		mockProvider.AssertExpectations(t)
	})
}

// TestGetMachineType tests the GetMachineType method
func TestGetMachineType(t *testing.T) {
	t.Parallel()
	exe := &Executable{}
	require.Equal(t, types.Risor, exe.GetMachineType())
}

// TestNewBytecodeEvaluator tests creating a new BytecodeEvaluator
func TestNewBytecodeEvaluator(t *testing.T) {
	t.Parallel()

	t.Run("with handler", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stderr, nil)
		exe := &script.ExecutableUnit{}

		evaluator := NewBytecodeEvaluator(handler, exe)

		require.NotNil(t, evaluator)
		require.Equal(t, constants.Ctx, evaluator.ctxKey)
		require.NotNil(t, evaluator.logger)
		require.Equal(t, handler, evaluator.logHandler)
	})

	t.Run("with nil handler", func(t *testing.T) {
		exe := &script.ExecutableUnit{}

		evaluator := NewBytecodeEvaluator(nil, exe)

		require.NotNil(t, evaluator)
		require.Equal(t, constants.Ctx, evaluator.ctxKey)
		require.NotNil(t, evaluator.logger)
		require.NotNil(t, evaluator.logHandler)
	})
}

// Helper function to create a test executable unit
func createTestExecutable(
	handler slog.Handler,
	ld loader.Loader,
	opt *RisorOptions,
	provider data.Provider,
) (*script.ExecutableUnit, error) {
	compiler := NewCompiler(handler, opt)
	reader, err := ld.GetReader()
	if err != nil {
		return nil, err
	}

	content, err := compiler.Compile(reader)
	if err != nil {
		return nil, err
	}

	return &script.ExecutableUnit{
		ID:           "test-id",
		Content:      content,
		DataProvider: provider,
	}, nil
}
