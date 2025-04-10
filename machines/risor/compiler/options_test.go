package compiler

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/stretchr/testify/require"
)

// TestCompilerOptionsDetailed tests all compiler options functionality in detail
func TestCompilerOptionsDetailed(t *testing.T) {
	t.Parallel()

	t.Run("Globals", func(t *testing.T) {
		t.Run("WithGlobals", func(t *testing.T) {
			t.Run("valid globals", func(t *testing.T) {
				globals := []string{"ctx", "print"}

				c := &Compiler{}
				c.applyDefaults()
				opt := WithGlobals(globals)
				err := opt(c)

				require.NoError(t, err)
				require.Equal(t, globals, c.globals)
			})

			t.Run("nil globals", func(t *testing.T) {
				c := &Compiler{}
				c.applyDefaults()
				nilOpt := WithGlobals(nil)
				err := nilOpt(c)

				require.NoError(t, err)
				require.Nil(t, c.globals)
			})

			t.Run("empty globals", func(t *testing.T) {
				c := &Compiler{}
				c.applyDefaults()
				emptyOpt := WithGlobals([]string{})
				err := emptyOpt(c)

				require.NoError(t, err)
				require.NotNil(t, c.globals)
				require.Empty(t, c.globals)
			})
		})

		t.Run("WithCtxGlobal", func(t *testing.T) {
			opt := WithCtxGlobal()

			t.Run("empty globals", func(t *testing.T) {
				c1 := &Compiler{globals: []string{}}
				err := opt(c1)

				require.NoError(t, err)
				require.Equal(t, []string{constants.Ctx}, c1.globals)
			})

			t.Run("existing globals without ctx", func(t *testing.T) {
				c2 := &Compiler{globals: []string{"request", "response"}}
				err := opt(c2)

				require.NoError(t, err)
				require.Equal(t, []string{"request", "response", constants.Ctx}, c2.globals)
			})

			t.Run("already contains ctx", func(t *testing.T) {
				c3 := &Compiler{globals: []string{constants.Ctx, "request"}}
				err := opt(c3)

				require.NoError(t, err)
				require.Equal(t, []string{constants.Ctx, "request"}, c3.globals)
				require.Len(t, c3.globals, 2) // Should not add duplicate
			})

			t.Run("nil globals", func(t *testing.T) {
				c4 := &Compiler{globals: nil}
				err := opt(c4)

				require.NoError(t, err)
				require.Equal(t, []string{constants.Ctx}, c4.globals)
			})
		})
	})

	t.Run("Logger", func(t *testing.T) {
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

		t.Run("option precedence", func(t *testing.T) {
			var handlerBuf, loggerBuf bytes.Buffer
			customHandler := slog.NewTextHandler(&handlerBuf, nil)
			customLogger := slog.New(slog.NewTextHandler(&loggerBuf, nil))

			t.Run("handler then logger", func(t *testing.T) {
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
			})

			// Clear buffers
			handlerBuf.Reset()
			loggerBuf.Reset()

			t.Run("logger then handler", func(t *testing.T) {
				c2, err := NewCompiler(
					WithLogger(customLogger),
					WithLogHandler(customHandler),
				)
				require.NoError(t, err)
				require.Equal(
					t,
					customHandler,
					c2.logHandler,
					"handler option should take precedence",
				)
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

		t.Run("WithLogHandler option", func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewTextHandler(&buf, nil)

			t.Run("valid handler", func(t *testing.T) {
				c := &Compiler{}
				c.applyDefaults()
				opt := WithLogHandler(handler)
				err := opt(c)

				require.NoError(t, err)
				require.Equal(t, handler, c.logHandler)
				require.Nil(t, c.logger) // Should clear Logger field
			})

			t.Run("nil handler", func(t *testing.T) {
				c := &Compiler{}
				c.applyDefaults()
				nilOpt := WithLogHandler(nil)
				err := nilOpt(c)

				require.Error(t, err)
				require.Contains(t, err.Error(), "log handler cannot be nil")
			})
		})

		t.Run("WithLogger option", func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewTextHandler(&buf, nil)
			logger := slog.New(handler)

			t.Run("valid logger", func(t *testing.T) {
				c := &Compiler{}
				c.applyDefaults()
				opt := WithLogger(logger)
				err := opt(c)

				require.NoError(t, err)
				require.Equal(t, logger, c.logger)
				require.Nil(t, c.logHandler) // Should clear LogHandler field
			})

			t.Run("nil logger", func(t *testing.T) {
				c := &Compiler{}
				c.applyDefaults()
				nilOpt := WithLogger(nil)
				err := nilOpt(c)

				require.Error(t, err)
				require.Contains(t, err.Error(), "logger cannot be nil")
			})
		})
	})

	t.Run("Defaults and Validation", func(t *testing.T) {
		t.Run("defaults - empty compiler", func(t *testing.T) {
			c := &Compiler{}
			c.applyDefaults()

			require.NotNil(t, c.logHandler)
			require.Nil(t, c.logger)
			require.NotNil(t, c.globals)
			require.Empty(t, c.globals)
		})

		t.Run("defaults - globals handling", func(t *testing.T) {
			t.Run("nil globals", func(t *testing.T) {
				c := &Compiler{
					globals: nil,
				}
				c.applyDefaults()

				require.NotNil(t, c.globals)
				require.Empty(t, c.globals)
			})

			t.Run("preserve non-nil globals", func(t *testing.T) {
				globals := []string{"test", "globals"}
				c := &Compiler{
					globals: globals,
				}
				c.applyDefaults()

				require.Equal(t, globals, c.globals)
			})

			t.Run("preserve empty globals", func(t *testing.T) {
				c := &Compiler{
					globals: []string{},
				}
				c.applyDefaults()

				require.NotNil(t, c.globals)
				require.Empty(t, c.globals)
			})
		})

		t.Run("validation", func(t *testing.T) {
			t.Run("valid compiler", func(t *testing.T) {
				c := &Compiler{}
				c.applyDefaults()

				err := c.validate()
				require.NoError(t, err)
			})

			t.Run("missing logger", func(t *testing.T) {
				c := &Compiler{}
				c.applyDefaults()
				c.logHandler = nil
				c.logger = nil

				err := c.validate()
				require.Error(t, err)
				require.Contains(t, err.Error(), "either log handler or logger must be specified")
			})

			t.Run("with log handler only", func(t *testing.T) {
				c := &Compiler{}
				c.logHandler = slog.NewTextHandler(bytes.NewBuffer(nil), nil)
				c.logger = nil

				err := c.validate()
				require.NoError(t, err)
			})

			t.Run("with logger only", func(t *testing.T) {
				c := &Compiler{}
				c.logHandler = nil
				c.logger = slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

				err := c.validate()
				require.NoError(t, err)
			})
		})
	})
}
