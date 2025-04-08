package risor

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithGlobals(t *testing.T) {
	// Test that WithGlobals properly sets the globals field
	globals := []string{"ctx", "print"}

	cfg := &compilerOptions{}
	opt := WithGlobals(globals)
	err := opt(cfg)

	require.NoError(t, err)
	require.Equal(t, globals, cfg.Globals)
}

func TestWithLogHandler(t *testing.T) {
	// Test that WithLogHandler properly sets the handler field
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)

	cfg := &compilerOptions{}
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

	cfg := &compilerOptions{}
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

func TestApplyDefaults(t *testing.T) {
	// Test that defaults are properly applied to an empty config
	cfg := &compilerOptions{}
	applyDefaults(cfg)

	require.NotNil(t, cfg.LogHandler)
	require.Nil(t, cfg.Logger)
	require.NotNil(t, cfg.Globals)
	require.Empty(t, cfg.Globals)
}

func TestValidate(t *testing.T) {
	// Test validation with empty config
	cfg := &compilerOptions{}
	applyDefaults(cfg)

	err := validate(cfg)
	require.NoError(t, err)

	// Test validation with manually cleared logger and handler
	cfg.LogHandler = nil
	cfg.Logger = nil

	err = validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "either log handler or logger must be specified")
}
