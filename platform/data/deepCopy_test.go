package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeepCopyMap checks the shape rules of deepCopyMap and deepCopyValue:
// the copy equals the source, nil containers stay nil rather than becoming
// empty, and a nil source map still yields a usable empty map.
func TestDeepCopyMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input map[string]any
		want  map[string]any
	}{
		{"nil map", nil, map[string]any{}},
		{"empty map", map[string]any{}, map[string]any{}},
		{"nested nil map stays nil", map[string]any{"m": map[string]any(nil)}, map[string]any{"m": map[string]any(nil)}},
		{"nested nil slice stays nil", map[string]any{"s": []any(nil)}, map[string]any{"s": []any(nil)}},
		{"empty slice stays empty", map[string]any{"s": []any{}}, map[string]any{"s": []any{}}},
		{"nested fixture", nestedFixtureHelper(), nestedFixtureHelper()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := deepCopyMap(tc.input)
			require.NotNil(t, got, "deepCopyMap should never return nil")
			assert.Equal(t, tc.want, got)

			if tc.input != nil {
				got["added"] = true
				assert.NotContains(t, tc.input, "added", "the copy should be a different map")
			}
		})
	}
}

// TestDeepCopyValue checks that container values are copied and that other
// values are returned unchanged.
func TestDeepCopyValue(t *testing.T) {
	t.Parallel()

	t.Run("nil slice is returned as a nil slice", func(t *testing.T) {
		got := deepCopyValue([]any(nil))
		s, ok := got.([]any)
		require.True(t, ok, "result should keep the []any type")
		assert.Nil(t, s, "a nil []any should not become an empty slice")
	})

	t.Run("slice is copied", func(t *testing.T) {
		src := []any{"a", map[string]any{"k": "v"}}
		got, ok := deepCopyValue(src).([]any)
		require.True(t, ok)
		require.Equal(t, src, got)

		got[0] = "changed"
		got[1].(map[string]any)["k"] = "changed"
		assert.Equal(t, []any{"a", map[string]any{"k": "v"}}, src, "source slice must be unchanged")
	})

	t.Run("typed slice is shared", func(t *testing.T) {
		src := []string{"a"}
		got, ok := deepCopyValue(src).([]string)
		require.True(t, ok)
		got[0] = "changed"
		assert.Equal(t, "changed", src[0], "typed slices are documented as shared with the source")
	})

	t.Run("scalar is returned as is", func(t *testing.T) {
		assert.Equal(t, 42, deepCopyValue(42))
		assert.Nil(t, deepCopyValue(nil))
	})
}
