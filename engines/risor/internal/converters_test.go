package internal

import (
	"testing"

	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRisorEnv(t *testing.T) {
	t.Parallel()

	t.Run("empty data includes builtins", func(t *testing.T) {
		env := BuildRisorEnv(constants.Ctx, map[string]any{})
		require.NotNil(t, env)
		// Should contain the ctx key
		_, hasCtx := env[constants.Ctx]
		assert.True(t, hasCtx, "env should contain the ctx key")
		// Should contain standard builtins like len, type, etc.
		_, hasLen := env["len"]
		assert.True(t, hasLen, "env should contain standard builtins")
	})

	t.Run("includes input data under ctx key", func(t *testing.T) {
		testData := map[string]any{"foo": "bar"}
		env := BuildRisorEnv(constants.Ctx, testData)
		require.NotNil(t, env)
		ctxData, ok := env[constants.Ctx].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "bar", ctxData["foo"])
	})
}
