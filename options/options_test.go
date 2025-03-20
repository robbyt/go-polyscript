package options

import (
	"io"
	"log/slog"
	"net/url"
	"os"
	"testing"

	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/types"
	"github.com/stretchr/testify/require"
)

// mockLoader is a simple implementation of loader.Loader for testing
type mockLoader struct{}

func (m *mockLoader) GetReader() (io.ReadCloser, error) {
	return nil, nil
}

func (m *mockLoader) GetSourceURL() *url.URL {
	u, _ := url.Parse("file:///mock")
	return u
}

func TestWithOptions(t *testing.T) {
	// Create test config
	cfg := &config{
		machineType: types.Starlark,
	}

	// Create test options
	testHandler := slog.NewTextHandler(os.Stdout, nil)
	testDataProvider := data.NewStaticProvider(map[string]any{"test": "value"})
	testLoader := &mockLoader{}

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
	cfg1 := &config{
		machineType: types.Starlark,
	}
	err := cfg1.validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no loader specified")

	// Test with missing machine type
	cfg2 := &config{
		loader: &mockLoader{},
	}
	err = cfg2.validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no machine type specified")

	// Test with valid config
	cfg3 := &config{
		machineType: types.Starlark,
		loader:      &mockLoader{},
	}
	err = cfg3.validate()
	require.NoError(t, err)
}

func TestConfigGetters(t *testing.T) {
	testHandler := slog.NewTextHandler(os.Stdout, nil)
	testDataProvider := data.NewStaticProvider(map[string]any{"test": "value"})
	testLoader := &mockLoader{}
	testCompilerOpts := "test-options"

	cfg := &config{
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
