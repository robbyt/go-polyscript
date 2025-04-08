package compiler

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	extismSDK "github.com/extism/go-sdk"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

func TestWithEntryPoint(t *testing.T) {
	// Test that WithEntryPoint properly sets the entry point
	entryPoint := "custom_entrypoint"

	cfg := &Options{}
	opt := WithEntryPoint(entryPoint)
	err := opt(cfg)

	require.NoError(t, err)
	require.Equal(t, entryPoint, cfg.EntryPoint)

	// Test with empty entry point
	emptyOpt := WithEntryPoint("")
	err = emptyOpt(cfg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "entry point cannot be empty")
}

func TestWithLogHandler(t *testing.T) {
	// Test that WithLogHandler properly sets the handler field
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)

	cfg := &Options{}
	opt := WithLogHandler(handler)
	err := opt(cfg)

	require.NoError(t, err)
	require.Equal(t, handler, cfg.LogHandler)
	require.Nil(t, cfg.Logger) // Should clear Logger field

	// Test with nil handler
	nilOpt := WithLogHandler(nil)
	err = nilOpt(cfg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "log handler cannot be nil")
}

func TestWithLogger(t *testing.T) {
	// Test that WithLogger properly sets the logger field
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)

	cfg := &Options{}
	opt := WithLogger(logger)
	err := opt(cfg)

	require.NoError(t, err)
	require.Equal(t, logger, cfg.Logger)
	require.Nil(t, cfg.LogHandler) // Should clear LogHandler field

	// Test with nil logger
	nilOpt := WithLogger(nil)
	err = nilOpt(cfg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "logger cannot be nil")
}

func TestWithWASIEnabled(t *testing.T) {
	// Test that WithWASIEnabled properly sets the EnableWASI field
	cfg := &Options{}

	// Test enabling WASI
	enableOpt := WithWASIEnabled(true)
	err := enableOpt(cfg)

	require.NoError(t, err)
	require.True(t, cfg.EnableWASI)

	// Test disabling WASI
	disableOpt := WithWASIEnabled(false)
	err = disableOpt(cfg)

	require.NoError(t, err)
	require.False(t, cfg.EnableWASI)
}

func TestWithRuntimeConfig(t *testing.T) {
	// Test that WithRuntimeConfig properly sets the RuntimeConfig field
	runtimeConfig := wazero.NewRuntimeConfig()

	cfg := &Options{}
	opt := WithRuntimeConfig(runtimeConfig)
	err := opt(cfg)

	require.NoError(t, err)
	require.Equal(t, runtimeConfig, cfg.RuntimeConfig)

	// Test with nil runtime config
	nilOpt := WithRuntimeConfig(nil)
	err = nilOpt(cfg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "runtime config cannot be nil")
}

func TestWithHostFunctions(t *testing.T) {
	// Test that WithHostFunctions properly sets the HostFunctions field
	testHostFn := extismSDK.NewHostFunctionWithStack(
		"test_function",
		func(ctx context.Context, p *extismSDK.CurrentPlugin, stack []uint64) {
			// No-op function for testing
		},
		nil, nil,
	)
	testHostFn.SetNamespace("test")

	hostFuncs := []extismSDK.HostFunction{testHostFn}

	cfg := &Options{}
	opt := WithHostFunctions(hostFuncs)
	err := opt(cfg)

	require.NoError(t, err)
	require.Equal(t, hostFuncs, cfg.HostFunctions)

	// Test with empty host functions
	emptyOpt := WithHostFunctions([]extismSDK.HostFunction{})
	err = emptyOpt(cfg)

	require.NoError(t, err)
	require.Empty(t, cfg.HostFunctions)
}

func TestApplyDefaults(t *testing.T) {
	// Test that defaults are properly applied to an empty config
	cfg := &Options{}
	ApplyDefaults(cfg)

	require.NotNil(t, cfg.LogHandler)
	require.Nil(t, cfg.Logger)
	require.Equal(t, defaultEntryPoint, cfg.EntryPoint)
	require.True(t, cfg.EnableWASI)
	require.NotNil(t, cfg.RuntimeConfig)
	require.NotNil(t, cfg.HostFunctions)
	require.Empty(t, cfg.HostFunctions)
}

func TestValidate(t *testing.T) {
	// Test validation with proper defaults
	cfg := &Options{}
	ApplyDefaults(cfg)

	err := Validate(cfg)
	require.NoError(t, err)

	// Test validation with manually cleared logger and handler
	cfg = &Options{}
	ApplyDefaults(cfg)
	cfg.LogHandler = nil
	cfg.Logger = nil

	err = Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "either log handler or logger must be specified")

	// Test validation with empty entry point
	cfg = &Options{}
	ApplyDefaults(cfg)
	cfg.EntryPoint = ""

	err = Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "entry point must be specified")

	// Test validation with nil runtime config
	cfg = &Options{}
	ApplyDefaults(cfg)
	cfg.RuntimeConfig = nil

	err = Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "runtime config cannot be nil")
}
