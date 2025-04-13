package evaluator

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	risorCompiler "github.com/risor-io/risor/compiler"
	"github.com/robbyt/go-polyscript/abstract/constants"
	"github.com/robbyt/go-polyscript/abstract/data"
	"github.com/robbyt/go-polyscript/abstract/script"
	"github.com/robbyt/go-polyscript/abstract/script/loader"
	"github.com/robbyt/go-polyscript/engines/risor/compiler"
	"github.com/robbyt/go-polyscript/engines/types"
	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/stretchr/testify/assert"
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

// Helper function to create a test executable unit
func createTestExecutable(
	handler slog.Handler,
	ld loader.Loader,
	globals []string,
	provider data.Provider,
) (*script.ExecutableUnit, error) {
	c, err := compiler.New(
		compiler.WithLogHandler(handler),
		compiler.WithGlobals(globals),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create compiler: %w", err)
	}

	reader, err := ld.GetReader()
	if err != nil {
		return nil, err
	}

	content, err := c.Compile(reader)
	if err != nil {
		return nil, err
	}

	return &script.ExecutableUnit{
		ID:           "test-id",
		Content:      content,
		DataProvider: provider,
	}, nil
}

// TestEvaluator_Evaluate tests evaluating Risor scripts
func TestEvaluator_Evaluate(t *testing.T) {
	t.Parallel()

	// Define a test script that handles HTTP requests
	testScript := `
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

	t.Run("success cases", func(t *testing.T) {
		tests := []struct {
			name           string
			script         string
			requestMethod  string
			urlPath        string
			expectedType   data.Types
			expectedResult string
			expectedValue  any
		}{
			{
				name:           "GET request to /hello",
				script:         testScript,
				requestMethod:  "GET",
				urlPath:        "/hello",
				expectedType:   data.Types("bool"),
				expectedResult: "true",
				expectedValue:  true,
			},
			{
				name:           "POST request",
				script:         testScript,
				requestMethod:  "POST",
				urlPath:        "/hello",
				expectedType:   data.Types("string"),
				expectedResult: "\"post\"",
				expectedValue:  "post",
			},
			{
				name:           "GET request to unknown path",
				script:         testScript,
				requestMethod:  "GET",
				urlPath:        "/unknown",
				expectedType:   data.Types("bool"),
				expectedResult: "false",
				expectedValue:  false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Set up the environment
				handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
					Level: slog.LevelDebug,
				})

				// Create the loader and provider
				ld, err := loader.NewFromString(tt.script)
				require.NoError(t, err)
				ctxProvider := data.NewContextProvider(constants.EvalData)

				// Create executable unit and evaluator
				exe, err := createTestExecutable(handler, ld, []string{constants.Ctx}, ctxProvider)
				require.NoError(t, err)
				evaluator := New(handler, exe)
				require.NotNil(t, evaluator)

				// Create the request data
				req := httptest.NewRequest(tt.requestMethod, tt.urlPath, nil)
				rMap, err := helpers.RequestToMap(req)
				require.NoError(t, err)
				require.NotNil(t, rMap)

				// Create the context with eval data
				evalData := map[string]any{
					constants.Request: rMap,
				}
				ctx := context.WithValue(context.Background(), constants.EvalData, evalData)

				// Execute the script
				response, err := evaluator.Eval(ctx)
				require.NoError(t, err)
				require.NotNil(t, response)

				// Verify the results
				require.Equal(t, tt.expectedType, response.Type())
				require.Equal(t, tt.expectedResult, response.Inspect())

				// Type-specific verification
				switch actualValue := response.Interface().(type) {
				case bool:
					expected, ok := tt.expectedValue.(bool)
					require.True(t, ok)
					require.Equal(t, expected, actualValue)
				case string:
					expected, ok := tt.expectedValue.(string)
					require.True(t, ok)
					require.Equal(t, expected, actualValue)
				default:
					require.Equal(t, tt.expectedValue, actualValue)
				}
			})
		}
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name         string
			setupExe     func() *script.ExecutableUnit
			errorMessage string
		}{
			{
				name: "nil executable unit",
				setupExe: func() *script.ExecutableUnit {
					return nil
				},
				errorMessage: "executable unit is nil",
			},
			{
				name: "nil bytecode",
				setupExe: func() *script.ExecutableUnit {
					return &script.ExecutableUnit{
						ID: "test-id",
						Content: &MockContent{
							Content: nil,
						},
					}
				},
				errorMessage: "bytecode is nil",
			},
			{
				name: "empty execution id",
				setupExe: func() *script.ExecutableUnit {
					return &script.ExecutableUnit{
						ID: "",
						Content: &MockContent{
							Content: &risorCompiler.Code{},
						},
					}
				},
				errorMessage: "exeID is empty",
			},
			{
				name: "wrong bytecode type",
				setupExe: func() *script.ExecutableUnit {
					return &script.ExecutableUnit{
						ID: "test-id",
						Content: &MockContent{
							Content: "not a risor bytecode",
						},
					}
				},
				errorMessage: "unable to type assert bytecode",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				handler := slog.NewTextHandler(os.Stderr, nil)
				exe := tt.setupExe()

				evaluator := &Evaluator{
					ctxKey:     constants.Ctx,
					execUnit:   exe,
					logHandler: handler,
					logger:     slog.New(handler),
				}

				ctx := context.Background()
				result, err := evaluator.Eval(ctx)

				require.Error(t, err)
				require.Nil(t, result)
				require.Contains(t, err.Error(), tt.errorMessage)
			})
		}
	})

	t.Run("load input data tests", func(t *testing.T) {
		tests := []struct {
			name         string
			setupExe     func() *script.ExecutableUnit
			setupCtx     func() context.Context
			expectError  bool
			errorMessage string
			expectEmpty  bool
		}{
			{
				name: "nil provider",
				setupExe: func() *script.ExecutableUnit {
					return nil
				},
				setupCtx: func() context.Context {
					return context.Background()
				},
				expectError: false,
				expectEmpty: true,
			},
			{
				name: "with provider error",
				setupExe: func() *script.ExecutableUnit {
					mockProvider := &MockProvider{}
					expectedErr := fmt.Errorf("provider error")
					mockProvider.On("GetData", mock.Anything).Return(nil, expectedErr)

					return &script.ExecutableUnit{
						DataProvider: mockProvider,
					}
				},
				setupCtx: func() context.Context {
					return context.Background()
				},
				expectError:  true,
				errorMessage: "provider error",
				expectEmpty:  true,
			},
			{
				name: "with empty data",
				setupExe: func() *script.ExecutableUnit {
					mockProvider := &MockProvider{}
					emptyData := map[string]any{}
					mockProvider.On("GetData", mock.Anything).Return(emptyData, nil)

					return &script.ExecutableUnit{
						DataProvider: mockProvider,
					}
				},
				setupCtx: func() context.Context {
					return context.Background()
				},
				expectError: false,
				expectEmpty: true,
			},
			{
				name: "with valid data",
				setupExe: func() *script.ExecutableUnit {
					mockProvider := &MockProvider{}
					validData := map[string]any{"test": "data"}
					mockProvider.On("GetData", mock.Anything).Return(validData, nil)

					return &script.ExecutableUnit{
						DataProvider: mockProvider,
					}
				},
				setupCtx: func() context.Context {
					return context.Background()
				},
				expectError: false,
				expectEmpty: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				handler := slog.NewTextHandler(os.Stderr, nil)
				exe := tt.setupExe()
				ctx := tt.setupCtx()

				evaluator := &Evaluator{
					ctxKey:     constants.Ctx,
					execUnit:   exe,
					logHandler: handler,
					logger:     slog.New(handler),
				}

				data, err := evaluator.loadInputData(ctx)

				if tt.expectError {
					require.Error(t, err)
					if tt.errorMessage != "" {
						require.Contains(t, err.Error(), tt.errorMessage)
					}
					require.Nil(t, data)
				} else {
					require.NoError(t, err)
					if tt.expectEmpty {
						assert.Empty(t, data)
					} else {
						assert.NotEmpty(t, data)
					}
				}

				// Verify mock expectations if we have a mockProvider
				if exe != nil && exe.DataProvider != nil {
					if mockProvider, ok := exe.DataProvider.(*MockProvider); ok {
						mockProvider.AssertExpectations(t)
					}
				}
			})
		}
	})

	t.Run("metadata tests", func(t *testing.T) {
		// Test String method
		t.Run("String method", func(t *testing.T) {
			evaluator := &Evaluator{}
			require.Equal(t, "risor.Evaluator", evaluator.String())
		})

		// Test constructor with various options
		t.Run("constructor options", func(t *testing.T) {
			tests := []struct {
				name        string
				handler     slog.Handler
				checkLogger bool
			}{
				{
					name:        "with handler",
					handler:     slog.NewTextHandler(os.Stderr, nil),
					checkLogger: true,
				},
				{
					name:        "with nil handler",
					handler:     nil,
					checkLogger: false,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					exe := &script.ExecutableUnit{}
					evaluator := New(tt.handler, exe)

					require.NotNil(t, evaluator)
					require.Equal(t, constants.Ctx, evaluator.ctxKey)
					require.NotNil(t, evaluator.logger)
					require.NotNil(t, evaluator.logHandler)

					if tt.checkLogger && tt.handler != nil {
						require.Equal(t, tt.handler, evaluator.logHandler)
					}
				})
			}
		})
	})
}

// TestEvaluator_PrepareContext tests the PrepareContext method with various scenarios
func TestEvaluator_PrepareContext(t *testing.T) {
	t.Parallel()

	// The test cases
	tests := []struct {
		name         string
		setupExe     func(t *testing.T) *script.ExecutableUnit
		inputs       []any
		wantError    bool
		errorMessage string
	}{
		{
			name: "with successful provider",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()

				mockProvider := &MockProvider{}
				enrichedCtx := context.WithValue(
					context.Background(),
					constants.EvalData,
					"enriched",
				)
				mockProvider.On("AddDataToContext", mock.Anything, mock.Anything).
					Return(enrichedCtx, nil)

				return &script.ExecutableUnit{DataProvider: mockProvider}
			},
			inputs:    []any{map[string]any{"test": "data"}},
			wantError: false,
		},
		{
			name: "with provider error",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()

				mockProvider := &MockProvider{}
				expectedErr := fmt.Errorf("provider error")
				mockProvider.On("AddDataToContext", mock.Anything, mock.Anything).
					Return(nil, expectedErr)

				return &script.ExecutableUnit{DataProvider: mockProvider}
			},
			inputs:       []any{map[string]any{"test": "data"}},
			wantError:    true,
			errorMessage: "provider error",
		},
		{
			name: "nil provider",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()
				return &script.ExecutableUnit{DataProvider: nil}
			},
			inputs:       []any{map[string]any{"test": "data"}},
			wantError:    true,
			errorMessage: "no data provider available",
		},
		{
			name: "nil executable unit",
			setupExe: func(t *testing.T) *script.ExecutableUnit {
				t.Helper()
				return nil
			},
			inputs:       []any{map[string]any{"test": "data"}},
			wantError:    true,
			errorMessage: "no data provider available",
		},
	}

	// Run the test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stderr, nil)
			exe := tt.setupExe(t)

			evaluator := &Evaluator{
				ctxKey:     constants.Ctx,
				execUnit:   exe,
				logHandler: handler,
				logger:     slog.New(handler),
			}

			ctx := context.Background()
			result, err := evaluator.PrepareContext(ctx, tt.inputs...)

			if tt.wantError {
				require.Error(t, err)
				if tt.errorMessage != "" {
					require.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}

			// If using mocks, verify expectations
			if exe != nil && exe.DataProvider != nil {
				if mockProvider, ok := exe.DataProvider.(*MockProvider); ok {
					mockProvider.AssertExpectations(t)
				}
			}
		})
	}
}
