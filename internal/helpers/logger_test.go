package helpers

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: this test mutates slog.Default via SetDefault and must not call
// t.Parallel(). Across packages it is safe because `go test ./...` runs
// each package in its own process (each with its own slog.Default).
func TestSetupLogger_NilHandlerInheritsDefault(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	custom := slog.NewTextHandler(&buf, nil)
	slog.SetDefault(slog.New(custom))

	h, logger := SetupLogger(nil, "myengine", "Compiler")
	require.NotNil(t, h, "handler should be non-nil after default fallback")
	require.NotNil(t, logger)

	logger.Info("hello", "k", "v")
	out := buf.String()
	assert.Contains(t, out, "hello", "message should flow to default handler")
	assert.Contains(t, out, "myengine.Compiler.k=v",
		"engine + group must prefix user attributes")
}

func TestSetupLogger_NonNilHandlerHonored(t *testing.T) {
	var buf bytes.Buffer
	custom := slog.NewTextHandler(&buf, nil)

	h, logger := SetupLogger(custom, "ignored-engine", "")
	require.NotNil(t, h)
	require.NotNil(t, logger)

	logger.Info("direct", "k", "v")
	out := buf.String()
	assert.Contains(t, out, "direct")
	assert.Contains(t, out, "k=v", "explicit handler should be used directly")
	assert.NotContains(t, out, "ignored-engine",
		"engine group must NOT be applied when handler is explicitly provided")
}

func TestSetupLogger_GroupNameAddsSubgroup(t *testing.T) {
	var buf bytes.Buffer
	custom := slog.NewTextHandler(&buf, nil)

	_, logger := SetupLogger(custom, "engine", "Compiler")
	logger.Info("msg", "k", "v")
	assert.Contains(t, buf.String(), "Compiler.k=v",
		"non-empty groupName must add a sub-group prefix")
}

func TestSetupLogger_EmptyGroupNameNoSubgroup(t *testing.T) {
	var buf bytes.Buffer
	custom := slog.NewTextHandler(&buf, nil)

	_, logger := SetupLogger(custom, "engine", "")
	logger.Info("msg", "k", "v")
	out := buf.String()
	assert.Contains(t, out, "k=v")
	assert.NotContains(t, out, "Compiler.")
}
