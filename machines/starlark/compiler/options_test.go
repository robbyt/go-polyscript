package compiler

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/stretchr/testify/require"
)

func TestWithGlobals(t *testing.T) {
	// Test that WithGlobals properly sets the globals field
	globals := []string{"ctx", "print"}

	c := &Compiler{}
	c.applyDefaults()
	opt := WithGlobals(globals)
	err := opt(c)

	require.NoError(t, err)
	require.Equal(t, globals, c.globals)

	// Test with nil globals
	c = &Compiler{}
	c.applyDefaults()
	nilOpt := WithGlobals(nil)
	err = nilOpt(c)

	require.NoError(t, err)
	require.Nil(t, c.globals)

	// Test with empty globals
	c = &Compiler{}
	c.applyDefaults()
	emptyOpt := WithGlobals([]string{})
	err = emptyOpt(c)

	require.NoError(t, err)
	require.NotNil(t, c.globals)
	require.Empty(t, c.globals)
}

func TestWithCtxGlobal(t *testing.T) {
	// Test with empty globals
	c1 := &Compiler{globals: []string{}}
	opt := WithCtxGlobal()
	err := opt(c1)

	require.NoError(t, err)
	require.Equal(t, []string{constants.Ctx}, c1.globals)

	// Test with existing globals not containing ctx
	c2 := &Compiler{globals: []string{"request", "response"}}
	err = opt(c2)

	require.NoError(t, err)
	require.Equal(t, []string{"request", "response", constants.Ctx}, c2.globals)

	// Test with globals already containing ctx
	c3 := &Compiler{globals: []string{constants.Ctx, "request"}}
	err = opt(c3)

	require.NoError(t, err)
	require.Equal(t, []string{constants.Ctx, "request"}, c3.globals)
	require.Len(t, c3.globals, 2) // Should not add duplicate

	// Test with nil globals
	c4 := &Compiler{globals: nil}
	err = opt(c4)

	require.NoError(t, err)
	require.Equal(t, []string{constants.Ctx}, c4.globals)
}

func TestLoggerConfiguration(t *testing.T) {
	t.Run("default initialization", func(t *testing.T) {
		// Create a compiler with default settings
		c, err := NewCompiler()
		require.NoError(t, err)

		// Verify that both logHandler and logger are set
		require.NotNil(t, c.logHandler, "logHandler should be initialized")
		require.NotNil(t, c.logger, "logger should be initialized")
	})

	t.Run("with explicit log handler", func(t *testing.T) {
		// Create a custom handler
		var buf bytes.Buffer
		customHandler := slog.NewTextHandler(&buf, nil)

		// Create compiler with the handler
		c, err := NewCompiler(WithLogHandler(customHandler))
		require.NoError(t, err)

		// Verify handler was set and used to create logger
		require.Equal(t, customHandler, c.logHandler, "custom handler should be set")
		require.NotNil(t, c.logger, "logger should be created from handler")

		// Test logging works with the custom handler
		c.logger.Info("test message")
		require.Contains(t, buf.String(), "test message", "log message should be in buffer")
	})

	t.Run("with explicit logger", func(t *testing.T) {
		// Create a custom logger
		var buf bytes.Buffer
		customHandler := slog.NewTextHandler(&buf, nil)
		customLogger := slog.New(customHandler)

		// Create compiler with the logger
		c, err := NewCompiler(WithLogger(customLogger))
		require.NoError(t, err)

		// Verify logger was set
		require.Equal(t, customLogger, c.logger, "custom logger should be set")
		require.NotNil(t, c.logHandler, "handler should be extracted from logger")

		// Test logging works with the custom logger
		c.logger.Info("test message")
		require.Contains(t, buf.String(), "test message", "log message should be in buffer")
	})

	t.Run("with both logger options, last one wins", func(t *testing.T) {
		// Create two buffers to verify which one receives logs
		var handlerBuf, loggerBuf bytes.Buffer
		customHandler := slog.NewTextHandler(&handlerBuf, nil)
		customLogger := slog.New(slog.NewTextHandler(&loggerBuf, nil))

		// Case 1: Handler then Logger
		c1, err := NewCompiler(
			WithLogHandler(customHandler),
			WithLogger(customLogger),
		)
		require.NoError(t, err)
		require.Equal(t, customLogger, c1.logger, "logger option should take precedence")
		c1.logger.Info("test message")
		require.Contains(t, loggerBuf.String(), "test message", "logger buffer should receive logs")
		require.Empty(t, handlerBuf.String(), "handler buffer should not receive logs")

		// Clear buffers
		handlerBuf.Reset()
		loggerBuf.Reset()

		// Case 2: Logger then Handler
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

func TestApplyDefaults(t *testing.T) {
	t.Run("empty compiler", func(t *testing.T) {
		// Test that defaults are properly applied to an empty compiler
		c := &Compiler{}
		c.applyDefaults()

		require.NotNil(t, c.logHandler)
		require.Nil(t, c.logger)
		require.NotNil(t, c.globals)
		require.Empty(t, c.globals)
	})

	t.Run("nil globals", func(t *testing.T) {
		// Test with a nil globals field
		c := &Compiler{
			globals: nil,
		}
		c.applyDefaults()

		require.NotNil(t, c.globals)
		require.Empty(t, c.globals)
	})

	t.Run("preserve non-nil globals", func(t *testing.T) {
		// Test that non-nil globals are preserved
		globals := []string{"test", "globals"}
		c := &Compiler{
			globals: globals,
		}
		c.applyDefaults()

		require.Equal(t, globals, c.globals)
	})

	t.Run("preserve empty globals", func(t *testing.T) {
		// Test that empty but non-nil globals are preserved
		c := &Compiler{
			globals: []string{},
		}
		c.applyDefaults()

		require.NotNil(t, c.globals)
		require.Empty(t, c.globals)
	})
}

func TestValidate(t *testing.T) {
	// Test validation with empty compiler after defaults
	c := &Compiler{}
	c.applyDefaults()

	err := c.validate()
	require.NoError(t, err)

	// Test validation with manually cleared logger and handler
	c.logHandler = nil
	c.logger = nil

	err = c.validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "either log handler or logger must be specified")

	// Test validation with either logger or handler
	c = &Compiler{}
	c.logHandler = slog.NewTextHandler(bytes.NewBuffer(nil), nil)
	c.logger = nil

	err = c.validate()
	require.NoError(t, err)

	c = &Compiler{}
	c.logHandler = nil
	c.logger = slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	err = c.validate()
	require.NoError(t, err)
}
