package starlark

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

	cfg := &compilerConfig{}
	opt := WithGlobals(globals)
	err := opt(cfg)

	require.NoError(t, err)
	require.Equal(t, globals, cfg.Globals)
}

func TestWithCtxGlobal(t *testing.T) {
	// Test with empty globals
	cfg1 := &compilerConfig{Globals: []string{}}
	opt := WithCtxGlobal()
	err := opt(cfg1)

	require.NoError(t, err)
	require.Equal(t, []string{constants.Ctx}, cfg1.Globals)

	// Test with existing globals not containing ctx
	cfg2 := &compilerConfig{Globals: []string{"request", "response"}}
	err = opt(cfg2)

	require.NoError(t, err)
	require.Equal(t, []string{"request", "response", constants.Ctx}, cfg2.Globals)

	// Test with globals already containing ctx
	cfg3 := &compilerConfig{Globals: []string{constants.Ctx, "request"}}
	err = opt(cfg3)

	require.NoError(t, err)
	require.Equal(t, []string{constants.Ctx, "request"}, cfg3.Globals)
	require.Len(t, cfg3.Globals, 2) // Should not add duplicate
}

func TestWithLogHandler(t *testing.T) {
	// Test that WithLogHandler properly sets the handler field
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)

	cfg := &compilerConfig{}
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

	cfg := &compilerConfig{}
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
	cfg := &compilerConfig{}
	applyDefaults(cfg)

	require.NotNil(t, cfg.LogHandler)
	require.Nil(t, cfg.Logger)
	require.NotNil(t, cfg.Globals)
	require.Empty(t, cfg.Globals)
}

func TestValidate(t *testing.T) {
	// Test validation with empty config
	cfg := &compilerConfig{}
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
