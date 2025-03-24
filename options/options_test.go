package options

import (
	"io"
	"log/slog"
	"net/url"
	"os"
	"testing"

	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockLoader is a testify mock implementation of loader.Loader for testing
type MockLoader struct {
	mock.Mock
}

func (m *MockLoader) GetReader() (io.ReadCloser, error) {
	args := m.Called()
	reader, _ := args.Get(0).(io.ReadCloser)
	return reader, args.Error(1)
}

func (m *MockLoader) GetSourceURL() *url.URL {
	args := m.Called()
	u, _ := args.Get(0).(*url.URL)
	return u
}

// NewMockLoader creates a pre-configured MockLoader with default expectations
func NewMockLoader() *MockLoader {
	mockLoader := new(MockLoader)

	// Set up default expectations
	mockLoader.On("GetReader").Return(nil, nil)

	u, err := url.Parse("file:///mock")
	if err != nil {
		panic(err) // This should never happen with a valid URL string
	}
	mockLoader.On("GetSourceURL").Return(u)

	return mockLoader
}

func TestWithOptions(t *testing.T) {
	// Create test config
	cfg := &Config{
		machineType: types.Starlark,
	}

	// Create test options
	testHandler := slog.NewTextHandler(os.Stdout, nil)
	testDataProvider := data.NewStaticProvider(map[string]any{"test": "value"})
	testLoader := NewMockLoader()

	// Create and apply options
	loggerOpt := WithLogger(testHandler)
	dataProviderOpt := WithDataProvider(testDataProvider)
	loaderOpt := WithLoader(testLoader)

	// Apply options
	err := loggerOpt(cfg)
	require.NoError(t, err)
	err = dataProviderOpt(cfg)
	require.NoError(t, err)
	err = loaderOpt(cfg)
	require.NoError(t, err)

	// Verify config was updated correctly
	require.Equal(t, testHandler, cfg.handler)
	require.Equal(t, testDataProvider, cfg.dataProvider)
	require.Equal(t, testLoader, cfg.loader)
}

func TestConfigValidation(t *testing.T) {
	// Test with missing loader
	cfg1 := &Config{
		machineType: types.Starlark,
	}
	err := cfg1.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no loader specified")

	// Test with missing machine type
	cfg2 := &Config{
		loader: NewMockLoader(),
	}
	err = cfg2.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no machine type specified")

	// Test with valid config
	cfg3 := &Config{
		machineType: types.Starlark,
		loader:      NewMockLoader(),
	}
	err = cfg3.Validate()
	require.NoError(t, err)
}

func TestConfigGetters(t *testing.T) {
	testHandler := slog.NewTextHandler(os.Stdout, nil)
	testDataProvider := data.NewStaticProvider(map[string]any{"test": "value"})
	testLoader := NewMockLoader()
	testCompilerOpts := "test-options"

	cfg := &Config{
		handler:         testHandler,
		machineType:     types.Starlark,
		dataProvider:    testDataProvider,
		loader:          testLoader,
		compilerOptions: testCompilerOpts,
	}

	require.Equal(t, testHandler, cfg.GetHandler())
	require.Equal(t, types.Starlark, cfg.GetMachineType())
	require.Equal(t, testDataProvider, cfg.GetDataProvider())
	require.Equal(t, testLoader, cfg.GetLoader())
	require.Equal(t, testCompilerOpts, cfg.GetCompilerOptions())
}
