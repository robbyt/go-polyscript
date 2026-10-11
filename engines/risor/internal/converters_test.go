package internal

import (
	"math"
	"testing"

	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRisorEnv(t *testing.T) {
	t.Parallel()

	t.Run("empty data includes builtins", func(t *testing.T) {
		env, err := BuildRisorEnv(constants.Ctx, map[string]any{})
		require.NoError(t, err)
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
		env, err := BuildRisorEnv(constants.Ctx, testData)
		require.NoError(t, err)
		require.NotNil(t, env)
		ctxData, ok := env[constants.Ctx].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "bar", ctxData["foo"])
	})
}

// TestBuildRisorEnv_UintRange is a regression for issue #165: Risor converts
// unsigned integers with int64(rv.Uint()), so values above math.MaxInt64
// would silently wrap negative. They must be rejected instead, wherever
// Risor's type registry would reach them.
func TestBuildRisorEnv_UintRange(t *testing.T) {
	t.Parallel()

	big := uint64(math.MaxUint64)
	inRange := uint64(math.MaxInt64)

	t.Run("out of range values are rejected", func(t *testing.T) {
		tests := []struct {
			name  string
			input map[string]any
		}{
			{"top-level uint64", map[string]any{"big": big}},
			{"top-level uint", map[string]any{"big": uint(math.MaxUint)}},
			{"top-level uintptr", map[string]any{"big": uintptr(math.MaxUint)}},
			{"nested map", map[string]any{"outer": map[string]any{"big": big}}},
			{"any slice", map[string]any{"list": []any{1, big}}},
			{"typed slice", map[string]any{"list": []uint64{1, big}}},
			{"array", map[string]any{"arr": [2]uint64{1, big}}},
			{"typed map", map[string]any{"m": map[string]uint64{"big": big}}},
			{"pointer", map[string]any{"ptr": &big}},
			{"deeply nested", map[string]any{
				"a": []any{map[string]any{"b": []map[string]any{{"c": big}}}},
			}},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				env, err := BuildRisorEnv(constants.Ctx, tc.input)
				require.ErrorIs(t, err, ErrUintOverflow)
				assert.Nil(t, env, "no env should be returned on overflow")
			})
		}
	})

	t.Run("error names the offending path", func(t *testing.T) {
		_, err := BuildRisorEnv(constants.Ctx, map[string]any{
			"outer": map[string]any{"list": []any{0, big}},
		})
		require.ErrorIs(t, err, ErrUintOverflow)
		assert.Contains(t, err.Error(), `ctx["outer"]["list"][1]`)
	})

	t.Run("in range values pass", func(t *testing.T) {
		tests := []struct {
			name  string
			input map[string]any
		}{
			{"max int64 as uint64", map[string]any{"v": inRange}},
			{"small uint", map[string]any{"v": uint(7)}},
			{"smaller unsigned kinds", map[string]any{
				"u8": uint8(math.MaxUint8), "u16": uint16(math.MaxUint16), "u32": uint32(math.MaxUint32),
			}},
			{"bytes", map[string]any{"b": []byte{0xff, 0xfe}}},
			{"nested in range", map[string]any{"m": map[string]any{"l": []uint64{inRange}}}},
			{"pointer in range", map[string]any{"p": &inRange}},
			{"nil values", map[string]any{"n": nil, "l": []any{nil}, "p": (*uint64)(nil)}},
			{"signed and other scalars", map[string]any{
				"i": int64(math.MinInt64), "f": 1.5, "s": "x", "b": true,
			}},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				env, err := BuildRisorEnv(constants.Ctx, tc.input)
				require.NoError(t, err)
				assert.Equal(t, tc.input, env[constants.Ctx], "input should be passed through unchanged")
			})
		}
	})
}
