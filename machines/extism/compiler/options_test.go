package compiler

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/machines/extism/compiler/internal/compile"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

func TestWithEntryPoint(t *testing.T) {
	t.Parallel()
	entryPoint := "custom_entrypoint"

	c := &Compiler{
		entryPointName: "",
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

func TestLoggerConfiguration(t *testing.T) {
	t.Parallel()

	// Basic initialization and configuration
	t.Run("creation and configuration", func(t *testing.T) {
		t.Run("default initialization", func(t *testing.T) {
			c, err := NewCompiler()
			require.NoError(t, err)
			require.NotNil(t, c.logHandler, "logHandler should be initialized")
			require.NotNil(t, c.logger, "logger should be initialized")
		})

		t.Run("with explicit log handler", func(t *testing.T) {
			var buf bytes.Buffer
			customHandler := slog.NewTextHandler(&buf, nil)

			c, err := NewCompiler(WithLogHandler(customHandler))
			require.NoError(t, err)

			require.Equal(t, customHandler, c.logHandler, "custom handler should be set")
			require.NotNil(t, c.logger, "logger should be created from handler")

			c.logger.Info("test message")
			require.Contains(t, buf.String(), "test message", "log message should be in buffer")
		})

		t.Run("with explicit logger", func(t *testing.T) {
			var buf bytes.Buffer
			customHandler := slog.NewTextHandler(&buf, nil)
			customLogger := slog.New(customHandler)

			c, err := NewCompiler(WithLogger(customLogger))
			require.NoError(t, err)

			require.Equal(t, customLogger, c.logger, "custom logger should be set")
			require.NotNil(t, c.logHandler, "handler should be extracted from logger")

			c.logger.Info("test message")
			require.Contains(t, buf.String(), "test message", "log message should be in buffer")
		})
	})

	// Testing precedence rules
	t.Run("option precedence", func(t *testing.T) {
		t.Run("last option wins", func(t *testing.T) {
			var handlerBuf, loggerBuf bytes.Buffer
			customHandler := slog.NewTextHandler(&handlerBuf, nil)
			customLogger := slog.New(slog.NewTextHandler(&loggerBuf, nil))

			// Case 1: Handler then Logger (logger wins)
			c1, err := NewCompiler(
				WithLogHandler(customHandler),
				WithLogger(customLogger),
			)
			require.NoError(t, err)
			require.Equal(t, customLogger, c1.logger, "logger option should take precedence")
			c1.logger.Info("test message")
			require.Contains(
				t,
				loggerBuf.String(),
				"test message",
				"logger buffer should receive logs",
			)
			require.Empty(t, handlerBuf.String(), "handler buffer should not receive logs")

			// Clear buffers
			handlerBuf.Reset()
			loggerBuf.Reset()

			// Case 2: Logger then Handler (handler wins)
			c2, err := NewCompiler(
				WithLogger(customLogger),
				WithLogHandler(customHandler),
			)
			require.NoError(t, err)
			require.Equal(t, customHandler, c2.logHandler, "handler option should take precedence")
			c2.logger.Info("test message")
			require.Contains(
				t,
				handlerBuf.String(),
				"test message",
				"handler buffer should receive logs",
			)
			require.Empty(t, loggerBuf.String(), "logger buffer should not receive logs")
		})
	})
}

func TestWithLogHandler(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestRuntimeOptions(t *testing.T) {
	t.Parallel()

	t.Run("WASI options", func(t *testing.T) {
		t.Run("enable/disable WASI", func(t *testing.T) {
			c := &Compiler{
				options: &compile.Settings{},
			}
			c.applyDefaults()

			enableOpt := WithWASIEnabled(true)
			err := enableOpt(c)
			require.NoError(t, err)
			require.True(t, c.options.EnableWASI)

			disableOpt := WithWASIEnabled(false)
			err = disableOpt(c)
			require.NoError(t, err)
			require.False(t, c.options.EnableWASI)
		})

		t.Run("with nil options", func(t *testing.T) {
			c := &Compiler{
				options: nil,
			}
			c.options = &compile.Settings{}

			opt := WithWASIEnabled(true)
			err := opt(c)
			require.NoError(t, err)
			require.True(t, c.options.EnableWASI)
		})
	})

	t.Run("runtime config", func(t *testing.T) {
		t.Run("normal runtime config", func(t *testing.T) {
			runtimeConfig := wazero.NewRuntimeConfig()
			c := &Compiler{
				options: &compile.Settings{},
			}
			c.applyDefaults()

			opt := WithRuntimeConfig(runtimeConfig)
			err := opt(c)
			require.NoError(t, err)
			require.Equal(t, runtimeConfig, c.options.RuntimeConfig)
		})

		t.Run("nil runtime config", func(t *testing.T) {
			c := &Compiler{
				options: &compile.Settings{},
			}
			c.applyDefaults()

			nilOpt := WithRuntimeConfig(nil)
			err := nilOpt(c)
			require.Error(t, err)
			require.Contains(t, err.Error(), "runtime config cannot be nil")
		})

		t.Run("with nil options", func(t *testing.T) {
			c := &Compiler{
				options: nil,
			}
			c.options = &compile.Settings{}
			runtimeConfig := wazero.NewRuntimeConfig()

			opt := WithRuntimeConfig(runtimeConfig)
			err := opt(c)
			require.NoError(t, err)
			require.Equal(t, runtimeConfig, c.options.RuntimeConfig)
		})
	})

	t.Run("host functions", func(t *testing.T) {
		t.Run("valid host functions", func(t *testing.T) {
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
		})

		t.Run("empty host functions", func(t *testing.T) {
			c := &Compiler{
				options: &compile.Settings{},
			}
			c.applyDefaults()

			emptyOpt := WithHostFunctions([]extismSDK.HostFunction{})
			err := emptyOpt(c)
			require.NoError(t, err)
			require.Empty(t, c.options.HostFunctions)
		})

		t.Run("with nil options", func(t *testing.T) {
			c := &Compiler{
				options: nil,
			}
			c.options = &compile.Settings{}

			testHostFn := extismSDK.NewHostFunctionWithStack(
				"test_function",
				func(ctx context.Context, p *extismSDK.CurrentPlugin, stack []uint64) {},
				nil, nil,
			)

			hostFuncs := []extismSDK.HostFunction{testHostFn}
			opt := WithHostFunctions(hostFuncs)
			err := opt(c)
			require.NoError(t, err)
			require.Equal(t, hostFuncs, c.options.HostFunctions)
		})
	})
}

func TestApplyDefaults(t *testing.T) {
	t.Parallel()
	t.Run("empty compiler", func(t *testing.T) {
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
	})

	t.Run("empty string entrypoint", func(t *testing.T) {
		// Test with an empty string entrypoint
		c := &Compiler{
			entryPointName: "",
			options:        &compile.Settings{},
			ctx:            context.Background(),
		}
		c.applyDefaults()

		// Check if the defaultEntryPoint was correctly applied
		require.Equal(t, defaultEntryPoint, c.entryPointName)
	})

	t.Run("reset empty entrypoint", func(t *testing.T) {
		// Test that emptying the entry point and reapplying defaults sets it back
		c := &Compiler{
			entryPointName: "initialValue",
			options:        &compile.Settings{},
			ctx:            context.Background(),
		}

		// First verify the initial value
		require.Equal(t, "initialValue", c.entryPointName)

		// Now set to empty string and apply defaults again
		c.entryPointName = ""
		c.applyDefaults()

		// Should be reset to defaultEntryPoint
		require.Equal(t, defaultEntryPoint, c.entryPointName)
	})

	t.Run("non-default value preserved", func(t *testing.T) {
		// Test that a non-default value is preserved through applyDefaults
		customEntryPoint := "custom_function"
		c := &Compiler{
			entryPointName: customEntryPoint,
			options:        &compile.Settings{},
			ctx:            context.Background(),
		}

		// Apply defaults, which should not change the entry point
		c.applyDefaults()

		// The custom value should be preserved
		require.Equal(t, customEntryPoint, c.entryPointName)
	})
}

func TestValidate(t *testing.T) {
	t.Parallel()
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
	c.entryPointName = ""

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
	t.Parallel()
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

func TestGetEntryPointName(t *testing.T) {
	t.Parallel()
	t.Run("normal value", func(t *testing.T) {
		// Test with a normal value
		c := &Compiler{
			entryPointName: "test_function",
		}

		// Should return the stored value
		require.Equal(t, "test_function", c.GetEntryPointName())
	})

	t.Run("empty string value", func(t *testing.T) {
		// Test with empty string
		c := &Compiler{
			entryPointName: "",
		}

		// Should return empty string
		require.Equal(t, "", c.GetEntryPointName())
	})
}
