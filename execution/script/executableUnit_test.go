package script

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	machineTypes "github.com/robbyt/go-polyscript/machines/types"
)

var emptyScriptData = make(map[string]any)

// Mock implementations
type mockLoader struct {
	mock.Mock
}

func (m *mockLoader) GetReader() (io.ReadCloser, error) {
	args := m.Called()
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *mockLoader) GetSourceURL() *url.URL {
	args := m.Called()
	return args.Get(0).(*url.URL)
}

type mockReadCloser struct {
	mock.Mock
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func (m *mockReadCloser) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestVersionMethods(t *testing.T) {
	t.Run("GetMachineType", func(t *testing.T) {
		mockContent := new(MockExecutableContent)
		expectedType := machineTypes.Risor
		mockContent.On("GetMachineType").Return(expectedType)

		exe := &ExecutableUnit{
			Content: mockContent,
		}

		machineType := exe.GetMachineType()
		require.Equal(t, expectedType, machineType, "Expected machine type to match")
		mockContent.AssertExpectations(t)
	})

	t.Run("GetCompiler", func(t *testing.T) {
		mockCompiler := new(MockCompiler)
		exe := &ExecutableUnit{
			Compiler: mockCompiler,
		}

		compiler := exe.GetCompiler()
		require.Equal(t, mockCompiler, compiler, "Expected compiler to match")
	})

	t.Run("GetContent", func(t *testing.T) {
		mockContent := new(MockExecutableContent)

		exe := &ExecutableUnit{
			Content: mockContent,
		}

		content := exe.GetContent()
		require.Equal(t, mockContent, content, "Expected content to match the mock content")
		mockContent.AssertExpectations(t)
	})

	t.Run("GetCreatedAt", func(t *testing.T) {
		createdAt := time.Now()
		exe := &ExecutableUnit{
			CreatedAt: createdAt,
		}

		timestamp := exe.GetCreatedAt()
		require.Equal(t, createdAt, timestamp, "Expected CreatedAt to match the provided timestamp")
	})
}

func TestNewVersion(t *testing.T) {
	t.Parallel()
	// Create test logger
	logHandler := slog.NewTextHandler(os.Stdout, nil)
	t.Run("ValidID", func(t *testing.T) {
		// Create script content and reader
		scriptContent := "print('Hello, World!')"

		lod, err := loader.NewFromString(scriptContent)
		require.NoError(t, err, "Expected no error when creating loader")

		reader, err := lod.GetReader()
		require.NoError(t, err, "Expected no error when getting reader")

		// Create mock loader instead of real loader
		mockLoader := new(mockLoader)
		mockLoader.On("GetReader").Return(reader, nil)

		// Setup mock compiler with same reader instance
		comp := new(MockCompiler)
		comp.On("Compile", reader).Return(&MockExecutableContent{}, nil)

		// Create executable unit
		exe, err := NewExecutableUnit(logHandler, t.Name(), mockLoader, comp, emptyScriptData)
		require.NoError(t, err, "Expected no error when creating executable unit")
		require.NotNil(t, exe, "Expected executable unit to be non-nil")
		require.Equal(t, t.Name(), exe.GetID(), "Expected ID to match")

		// Verify all mocks
		mockLoader.AssertExpectations(t)
		comp.AssertExpectations(t)
	})

	t.Run("ValidContent", func(t *testing.T) {
		scriptBody := "valid script content"
		lod, err := loader.NewFromString(scriptBody)
		require.NoError(t, err, "Expected no error when creating a new loader with valid content")

		reader, err := lod.GetReader()
		require.NoError(t, err, "Expected no error when getting reader")

		comp := new(MockCompiler)
		mockContent := new(MockExecutableContent)
		comp.On("Compile", reader).Return(mockContent, nil).Once()

		exe, err := NewExecutableUnit(logHandler, t.Name(), lod, comp, emptyScriptData)
		require.NoError(t, err, "Expected no error when creating a new version with valid content")
		require.NotNil(t, exe, "Expected version to be non-nil")
		require.Equal(t, mockContent, exe.GetContent(), "Expected content to match the mock content")
		require.NotNil(t, exe.GetLoader().GetSourceURL(), "Expected SourceURI to be non-nil")
		require.Contains(t, exe.GetLoader().GetSourceURL().String(), "string://inline/")
		require.WithinDuration(t, time.Now(), exe.GetCreatedAt(), time.Second, "Expected CreatedAt to be within the last second")

		comp.AssertExpectations(t)
		mockContent.AssertExpectations(t)
	})

	t.Run("ValidationError", func(t *testing.T) {
		scriptBody := "invalid script content"
		lod, err := loader.NewFromString(scriptBody)
		require.NoError(t, err, "Expected no error when creating a new loader with empty content")

		reader, err := lod.GetReader()
		require.NoError(t, err, "Expected no error when getting reader")

		comp := new(MockCompiler)
		validationError := errors.New("validation failed")
		comp.On("Compile", reader).Return(nil, validationError).Once()

		exe, err := NewExecutableUnit(logHandler, t.Name(), lod, comp, emptyScriptData)
		require.Error(t, err)
		require.Nil(t, exe)
		require.ErrorIs(t, err, validationError)

		comp.AssertExpectations(t)
	})

	t.Run("EmptyVersionID_ReturnsChecksum", func(t *testing.T) {
		// Create script content
		scriptContent := "test content"

		// Create loader and get reader
		lod, err := loader.NewFromString(scriptContent)
		require.NoError(t, err, "Expected no error when creating loader")

		reader, err := lod.GetReader()
		require.NoError(t, err, "Expected no error when getting reader")

		// Create mock loader instead of real loader
		mockLoader := new(mockLoader)
		mockLoader.On("GetReader").Return(reader, nil)

		// Setup mock compiler with same reader instance
		mockCompiler := new(MockCompiler)
		mockContent := new(MockExecutableContent)
		// Add expectation for GetSource
		mockContent.On("GetSource").Return(scriptContent)
		mockCompiler.On("Compile", reader).Return(mockContent, nil)

		// Create executable unit with empty ID
		exe, err := NewExecutableUnit(logHandler, "", mockLoader, mockCompiler, emptyScriptData)
		require.NoError(t, err)
		require.NotNil(t, exe)

		// Verify version ID is set from content checksum
		versionID := exe.GetID()
		require.NotEmpty(t, versionID, "Expected version ID to be non-empty")
		require.Equal(t, len(versionID), checksumLength, "Expected version ID length to match checksum length")

		// Verify all mocks
		mockLoader.AssertExpectations(t)
		mockCompiler.AssertExpectations(t)
		mockContent.AssertExpectations(t)
	})

	t.Run("NilCompiler", func(t *testing.T) {
		exe, err := NewExecutableUnit(logHandler, "test", &mockLoader{}, nil, emptyScriptData)
		require.Error(t, err)
		require.Nil(t, exe)
		require.Contains(t, err.Error(), "compiler is nil")
	})

	t.Run("EmptyContent", func(t *testing.T) {
		loader, err := loader.NewFromString("")

		require.Nil(t, loader)
		require.Error(t, err)
	})

	t.Run("EmptyContentFromReader", func(t *testing.T) {
		// Setup mock reader with proper expectations
		mockReader := new(mockReadCloser)

		// Setup mock loader with source URL
		mockLoader := new(mockLoader)
		mockLoader.On("GetReader").Return(mockReader, nil)

		// Setup mock compiler with expected error
		mockCompiler := new(MockCompiler)
		mockCompiler.On("Compile", mockReader).Return(nil, errors.New("empty content"))
		// Create executable unit
		exe, err := NewExecutableUnit(logHandler, "test", mockLoader, mockCompiler, emptyScriptData)
		require.Error(t, err)
		require.Nil(t, exe)

		// Verify all mocks
		mockReader.AssertExpectations(t)
		mockLoader.AssertExpectations(t)
		mockCompiler.AssertExpectations(t)
	})

	t.Run("GetReaderError", func(t *testing.T) {
		mockReader := new(mockReadCloser)

		mockLoader := new(mockLoader)
		mockLoader.On("GetReader").Return(mockReader, errors.New("get reader error")).Once()

		exe, err := NewExecutableUnit(logHandler, "test", mockLoader, new(MockCompiler), emptyScriptData)
		require.Error(t, err)
		require.Nil(t, exe)

		mockReader.AssertExpectations(t)
		mockLoader.AssertExpectations(t)
	})

	t.Run("ReaderError", func(t *testing.T) {
		// Setup mock reader with read error
		mockReader := new(mockReadCloser)

		// Setup mock loader
		mockLoader := new(mockLoader)
		mockLoader.On("GetReader").Return(mockReader, nil).Once()

		// Setup mock compiler with same reader instance
		mockCompiler := new(MockCompiler)
		mockCompiler.On("Compile", mockReader).Return(nil, errors.New("compile failed")).Once()

		// Create executable unit
		exe, err := NewExecutableUnit(logHandler, "test", mockLoader, mockCompiler, emptyScriptData)
		require.Error(t, err)
		require.Nil(t, exe)

		// Verify all mocks
		mockReader.AssertExpectations(t)
		mockLoader.AssertExpectations(t)
		mockCompiler.AssertExpectations(t)
	})
}

func TestExecutableUnit_String(t *testing.T) {
	t.Parallel()

	t.Run("String method", func(t *testing.T) {
		mockLoader := new(mockLoader)
		mockCompiler := new(MockCompiler)
		mockContent := new(MockExecutableContent)

		exe := &ExecutableUnit{
			ID:        "testID",
			CreatedAt: time.Now(),
			Loader:    mockLoader,
			Content:   mockContent,
			Compiler:  mockCompiler,
		}

		expectedString := fmt.Sprintf("ExecutableUnit{ID: %s, CreatedAt: %s, Compiler: %s, Loader: %s}",
			exe.ID, exe.CreatedAt, exe.Compiler, exe.Loader)

		require.Equal(t, expectedString, exe.String(), "Expected string representation to match")
	})
}

func TestNewVersionWithScriptData(t *testing.T) {
	t.Parallel()
	// Create test logger
	logHandler := slog.NewTextHandler(os.Stdout, nil)

	t.Run("ValidScriptData", func(t *testing.T) {
		// Script content
		scriptContent := "print('Hello, World!')"

		// Create mock compiler and mock content
		mockCompiler := new(MockCompiler)
		mockContent := new(MockExecutableContent)

		// Create lod and set mock expectations
		lod, err := loader.NewFromString(scriptContent)
		require.NoError(t, err, "Expected no error when creating loader")

		reader, err := lod.GetReader()
		require.NoError(t, err, "Expected no error when getting reader")

		mockCompiler.On("Compile", reader).Return(mockContent, nil).Once()

		// Create loader directly from string instead of file
		loader, err := loader.NewFromString(scriptContent)
		require.NoError(t, err, "Expected no error when creating loader")

		// Test data
		scriptData := map[string]any{
			"key1": "value1",
			"key2": 42,
		}

		// Create executable unit
		exe, err := NewExecutableUnit(logHandler, t.Name(), loader, mockCompiler, scriptData)
		require.NoError(t, err, "Expected no error creating executable unit")
		require.NotNil(t, exe, "Expected executable unit to be non-nil")
		require.Equal(t, scriptData, exe.GetScriptData(), "Expected script data to match")

		// Verify all mocks
		mockCompiler.AssertExpectations(t)
		mockContent.AssertExpectations(t)
	})

	t.Run("EmptyScriptData", func(t *testing.T) {
		scriptBody := "valid script content"

		lod, err := loader.NewFromString(scriptBody)
		require.NoError(t, err, "Expected no error when creating a new loader with valid content")

		reader, err := lod.GetReader()
		require.NoError(t, err, "Expected no error when getting reader")

		comp := new(MockCompiler)
		mockContent := new(MockExecutableContent)

		comp.On("Compile", reader).Return(mockContent, nil).Once()

		exe, err := NewExecutableUnit(logHandler, t.Name(), lod, comp, nil)
		require.NoError(t, err, "Expected no error when creating a new version with nil script data")
		require.NotNil(t, exe, "Expected version to be non-nil")
		require.Empty(t, exe.GetScriptData(), "Expected script data to be empty")

		comp.AssertExpectations(t)
		mockContent.AssertExpectations(t)
	})
}

func TestExecutableUnit_BuildEvalContext(t *testing.T) {
	t.Run("request data handling", func(t *testing.T) {
		tests := []struct {
			name       string
			request    *http.Request
			scriptData map[string]any
			wantData   map[string]any
		}{
			{
				name: "valid request with headers",
				request: func() *http.Request {
					req := httptest.NewRequest("GET", "/test", nil)
					req.Header.Add("Content-Type", "application/json")
					req.RemoteAddr = "192.0.2.1:1234"
					return req
				}(),
				scriptData: map[string]any{"config": "value"},
				wantData: map[string]any{
					constants.Request: map[string]any{
						"Method":        "GET",
						"URL":           &url.URL{Path: "/test"},
						"URL_String":    "/test",
						"URL_Host":      "",
						"URL_Scheme":    "",
						"URL_Path":      "/test",
						"Proto":         "HTTP/1.1",
						"Headers":       map[string][]string{"Content-Type": {"application/json"}},
						"Body":          "",
						"ContentLength": int64(0),
						"Host":          "example.com",
						"RemoteAddr":    "192.0.2.1:1234",
						"QueryParams":   map[string][]string{},
					},
					constants.ScriptData: map[string]any{
						"config": "value",
					},
				},
			},
			{
				name:       "nil script data",
				request:    httptest.NewRequest("POST", "/api", nil),
				scriptData: nil,
				wantData: map[string]any{
					constants.Request: map[string]any{
						"Method":        "POST",
						"URL":           &url.URL{Path: "/api"},
						"URL_String":    "/api",
						"URL_Host":      "",
						"URL_Scheme":    "",
						"URL_Path":      "/api",
						"Proto":         "HTTP/1.1",
						"Headers":       map[string][]string{},
						"Body":          "",
						"ContentLength": int64(0),
						"Host":          "example.com",
						"RemoteAddr":    "192.0.2.1:1234",
						"QueryParams":   map[string][]string{},
					},
					constants.ScriptData: map[string]any{},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				unit := &ExecutableUnit{
					ID:         "test-unit",
					ScriptData: tt.scriptData,
				}

				ctx := context.Background()
				resultCtx := unit.BuildEvalContext(ctx, tt.request)

				// Get the eval data from context
				evalData, ok := resultCtx.Value(constants.EvalData).(map[string]any)
				require.True(t, ok, "Expected eval data in context")

				// Check request data
				reqData, ok := evalData[constants.Request].(map[string]any)
				require.True(t, ok, "Expected request data in eval data")
				require.Equal(t, tt.wantData[constants.Request], reqData)

				// Check script data
				scriptData, ok := evalData[constants.ScriptData].(map[string]any)
				require.True(t, ok, "Expected script data in eval data")
				require.Equal(t, tt.wantData[constants.ScriptData], scriptData)
			})
		}
	})

	t.Run("request conversion error", func(t *testing.T) {
		// Create a test logger
		logHandler := slog.NewTextHandler(os.Stdout, nil)
		logger := slog.New(logHandler.WithGroup("test"))

		unit := &ExecutableUnit{
			ID:         "test-unit",
			logHandler: logHandler,
			logger:     logger,
		}

		// Create invalid request that won't panic but will cause conversion error
		req := &http.Request{
			Method: "GET",
			URL: &url.URL{
				// Invalid URL schema that will cause conversion error but not panic
				Scheme: "%%%",
				Path:   "/test",
			},
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        make(http.Header),
			ContentLength: 0,
		}

		ctx := context.Background()
		resultCtx := unit.BuildEvalContext(ctx, req)

		// Get the eval data from context
		evalData, ok := resultCtx.Value(constants.EvalData).(map[string]any)
		require.True(t, ok, "Expected eval data in context")

		// Check that request data is an empty map on error
		reqData, ok := evalData[constants.Request].(map[string]any)
		require.True(t, ok, "Expected request data in eval data")
		require.Empty(t, reqData, "Expected empty request data on error")

		// Verify script data is still present even with request error
		scriptData, ok := evalData[constants.ScriptData].(map[string]any)
		require.True(t, ok, "Expected script data in eval data")
		require.Empty(t, scriptData, "Expected empty script data")
	})
}
