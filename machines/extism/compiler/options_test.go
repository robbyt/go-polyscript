package compiler

import (
	"bytes"
	"context"
	"log/slog"
	"sync/atomic"
	"testing"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/machines/extism/compiler/internal/compile"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

func TestWithEntryPoint(t *testing.T) {
	// Test that WithEntryPoint properly sets the entry point
	entryPoint := "custom_entrypoint"

	c := &Compiler{
		entryPointName: atomic.Value{},
	}
	c.applyDefaults()
	opt := WithEntryPoint(entryPoint)
	err := opt(c)

	require.NoError(t, err)
	require.Equal(t, entryPoint, c.GetEntryPointName())

	// Test with empty entry point
	emptyOpt := WithEntryPoint("")
	err = emptyOpt(c)

	require.Error(t, err)
	require.Contains(t, err.Error(), "entry point cannot be empty")
}

func TestWithLogHandler(t *testing.T) {
	// Test that WithLogHandler properly sets the handler field
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)

	c := &Compiler{}
	c.applyDefaults()
	opt := WithLogHandler(handler)
	err := opt(c)

	require.NoError(t, err)
	require.Equal(t, handler, c.logHandler)
	require.Nil(t, c.logger) // Should clear Logger field

	// Test with nil handler
	nilOpt := WithLogHandler(nil)
	err = nilOpt(c)

	require.Error(t, err)
	require.Contains(t, err.Error(), "log handler cannot be nil")
}

func TestWithLogger(t *testing.T) {
	// Test that WithLogger properly sets the logger field
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)

	c := &Compiler{}
	c.applyDefaults()
	opt := WithLogger(logger)
	err := opt(c)

	require.NoError(t, err)
	require.Equal(t, logger, c.logger)
	require.Nil(t, c.logHandler) // Should clear LogHandler field

	// Test with nil logger
	nilOpt := WithLogger(nil)
	err = nilOpt(c)

	require.Error(t, err)
	require.Contains(t, err.Error(), "logger cannot be nil")
}

func TestWithWASIEnabled(t *testing.T) {
	// Test that WithWASIEnabled properly sets the EnableWASI field
	c := &Compiler{
		options: &compile.Settings{},
	}
	c.applyDefaults()

	// Test enabling WASI
	enableOpt := WithWASIEnabled(true)
	err := enableOpt(c)

	require.NoError(t, err)
	require.True(t, c.options.EnableWASI)

	// Test disabling WASI
	disableOpt := WithWASIEnabled(false)
	err = disableOpt(c)

	require.NoError(t, err)
	require.False(t, c.options.EnableWASI)
}

func TestWithRuntimeConfig(t *testing.T) {
	// Test that WithRuntimeConfig properly sets the RuntimeConfig field
	runtimeConfig := wazero.NewRuntimeConfig()

	c := &Compiler{
		options: &compile.Settings{},
	}
	c.applyDefaults()
	opt := WithRuntimeConfig(runtimeConfig)
	err := opt(c)

	require.NoError(t, err)
	require.Equal(t, runtimeConfig, c.options.RuntimeConfig)

	// Test with nil runtime config
	nilOpt := WithRuntimeConfig(nil)
	err = nilOpt(c)

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

	c := &Compiler{
		options: &compile.Settings{},
	}
	c.applyDefaults()
	opt := WithHostFunctions(hostFuncs)
	err := opt(c)

	require.NoError(t, err)
	require.Equal(t, hostFuncs, c.options.HostFunctions)

	// Test with empty host functions
	emptyOpt := WithHostFunctions([]extismSDK.HostFunction{})
	err = emptyOpt(c)

	require.NoError(t, err)
	require.Empty(t, c.options.HostFunctions)
}

func TestApplyDefaults(t *testing.T) {
	// Test that defaults are properly applied to an empty compiler
	c := &Compiler{}
	c.applyDefaults()

	require.NotNil(t, c.logHandler)
	require.Nil(t, c.logger)
	require.Equal(t, defaultEntryPoint, c.GetEntryPointName())
	require.NotNil(t, c.options)
	require.True(t, c.options.EnableWASI)
	require.NotNil(t, c.options.RuntimeConfig)
	require.NotNil(t, c.options.HostFunctions)
	require.Empty(t, c.options.HostFunctions)
	require.NotNil(t, c.ctx)
}

func TestValidate(t *testing.T) {
	// Test validation with proper defaults
	c := &Compiler{}
	c.applyDefaults()

	err := c.validate()
	require.NoError(t, err)

	// Test validation with manually cleared logger and handler
	c = &Compiler{}
	c.applyDefaults()
	c.logHandler = nil
	c.logger = nil

	err = c.validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "either log handler or logger must be specified")

	// Test validation with empty entry point
	c = &Compiler{}
	c.applyDefaults()
	c.entryPointName.Store("")

	err = c.validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "entry point must be specified")

	// Test validation with nil runtime config
	c = &Compiler{}
	c.applyDefaults()
	c.options.RuntimeConfig = nil

	err = c.validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "runtime config cannot be nil")
}

func TestWithContext(t *testing.T) {
	// Test that WithContext properly sets the Context field
	ctx := context.Background()

	c := &Compiler{}
	c.applyDefaults()
	opt := WithContext(ctx)
	err := opt(c)

	require.NoError(t, err)
	require.Equal(t, ctx, c.ctx)

	// We need to test our validation of nil contexts but without passing nil directly
	// to satisfy the linter. Use a type conversion trick to create a nil context.
	var nilContext context.Context
	nilOpt := WithContext(nilContext)
	err = nilOpt(c)

	require.Error(t, err)
	require.Contains(t, err.Error(), "context cannot be nil")
}
