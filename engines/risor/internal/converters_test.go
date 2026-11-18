package internal

import (
	"math"
	"reflect"
	"testing"

	"github.com/deepnoodle-ai/risor/v2/pkg/object"
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

// safeValuer converts itself to a Risor string, so Risor never reads its
// oversized field.
type safeValuer struct{ Big uint64 }

func (safeValuer) RisorValue() object.Object { return object.NewString("safe") }

// safeUint is an unsigned type that converts itself instead of being read as
// an integer.
type safeUint uint64

func (safeUint) RisorValue() object.Object { return object.NewString("safe") }

// ptrValuer converts itself only through its pointer, so a ptrValuer stored
// by value is wrapped as a GoStruct and its fields are read.
type ptrValuer struct{ Big uint64 }

func (*ptrValuer) RisorValue() object.Object { return object.NewString("safe") }

// TestBuildRisorEnv_UintRange is a regression for issue #165: Risor converts
// unsigned integers with int64(rv.Uint()), so values above math.MaxInt64
// would silently wrap negative. They must be rejected instead, wherever
// Risor's type registry would reach them.
func TestBuildRisorEnv_UintRange(t *testing.T) {
	t.Parallel()

	big := uint64(math.MaxUint64)
	inRange := uint64(math.MaxInt64)

	type inner struct{ Big uint64 }
	type outer struct {
		Name   string
		Inner  inner
		Ptr    *inner
		Nest   map[string]any
		hidden uint64 // unexported: Risor cannot read it, so it is not checked
	}
	// Risor does not promote fields through embedding: the fields of an
	// unexported embedded struct are not exposed as attributes, while an
	// exported embedded struct is a field named after its type.
	type Shown struct{ Big uint64 }
	type hidesEmbedded struct{ inner }
	type hidesEmbeddedPtr struct{ *inner }
	type showsEmbedded struct{ Shown }
	// node forms a pointer cycle, which Risor reads lazily without trouble.
	type node struct {
		Val  uint64
		Next *node
	}
	cycle := &node{Val: 1}
	cycle.Next = cycle
	badCycle := &node{Val: 1}
	badCycle.Next = &node{Val: big, Next: badCycle}

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
			{"struct field", map[string]any{"s": inner{Big: big}}},
			{"struct pointer field", map[string]any{"s": &inner{Big: big}}},
			{"nested struct field", map[string]any{"s": outer{Inner: inner{Big: big}}}},
			{"struct pointer in struct", map[string]any{"s": outer{Ptr: &inner{Big: big}}}},
			{"map in struct field", map[string]any{"s": outer{Nest: map[string]any{"x": big}}}},
			{"slice of structs", map[string]any{"l": []inner{{Big: 1}, {Big: big}}}},
			{"struct in pointer cycle", map[string]any{"n": badCycle}},
			{"pointer-receiver valuer stored by value", map[string]any{"v": ptrValuer{Big: big}}},
			{"exported embedded struct", map[string]any{"s": showsEmbedded{Shown{Big: big}}}},
			{"exported embedded struct by pointer", map[string]any{"s": &showsEmbedded{Shown{Big: big}}}},
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

		_, err = BuildRisorEnv(constants.Ctx, map[string]any{
			"s": outer{Nest: map[string]any{"x": big}},
		})
		require.ErrorIs(t, err, ErrUintOverflow)
		assert.Contains(t, err.Error(), `ctx["s"].Nest["x"]`)

		_, err = BuildRisorEnv(constants.Ctx, map[string]any{
			"s": showsEmbedded{Shown{Big: big}},
		})
		require.ErrorIs(t, err, ErrUintOverflow)
		assert.Contains(t, err.Error(), `ctx["s"].Shown.Big`)
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
			{"struct in range", map[string]any{"s": &outer{
				Name: "x", Inner: inner{Big: inRange}, Nest: map[string]any{"x": inRange},
			}}},
			{"unexported struct field is not checked", map[string]any{"s": outer{hidden: big}}},
			{"unexported embedded struct is not checked", map[string]any{"s": hidesEmbedded{inner{Big: big}}}},
			{"unexported embedded pointer is not checked", map[string]any{"s": hidesEmbeddedPtr{&inner{Big: big}}}},
			{"pointer to unexported embedded struct is not checked", map[string]any{
				"s": &hidesEmbedded{inner{Big: big}},
			}},
			{"nil struct pointer", map[string]any{"s": (*outer)(nil)}},
			{"pointer cycle in range", map[string]any{"n": cycle}},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				env, err := BuildRisorEnv(constants.Ctx, tc.input)
				require.NoError(t, err)
				assert.Equal(t, tc.input, env[constants.Ctx], "input should be passed through unchanged")
			})
		}
	})

	t.Run("values risor converts itself are skipped", func(t *testing.T) {
		tests := []struct {
			name  string
			input map[string]any
		}{
			{"valuer struct with big field", map[string]any{"v": safeValuer{Big: big}}},
			{"valuer struct pointer with big field", map[string]any{"v": &safeValuer{Big: big}}},
			{"pointer-receiver valuer by pointer", map[string]any{"v": &ptrValuer{Big: big}}},
			{"named uint valuer", map[string]any{"v": safeUint(big)}},
			{"map of named uint valuers", map[string]any{"m": map[string]safeUint{"x": safeUint(big)}}},
			{"slice of named uint valuers", map[string]any{"l": []safeUint{safeUint(big)}}},
			{"valuer in struct field", map[string]any{"s": struct{ V safeValuer }{safeValuer{Big: big}}}},
			{"risor object", map[string]any{"o": object.NewInt(1)}},
			{"map of risor objects", map[string]any{"m": map[string]object.Object{"x": object.NewInt(1)}}},
			{"nil risor object in struct field", map[string]any{"s": struct{ O object.Object }{}}},
			{"typed-nil valuer pointer", map[string]any{"v": (*safeValuer)(nil)}},
			{"typed-nil pointer-receiver valuer", map[string]any{"v": (*ptrValuer)(nil)}},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				env, err := BuildRisorEnv(constants.Ctx, tc.input)
				require.NoError(t, err)
				assert.Equal(t, tc.input, env[constants.Ctx], "input should be passed through unchanged")
			})
		}
	})

	t.Run("risor does not promote embedded fields", func(t *testing.T) {
		// Pins the Risor behaviour checkStruct relies on: if Risor ever
		// promotes embedded fields, the unexported embedded cases above
		// would let an oversized field through.
		registry := object.DefaultRegistry()
		for name, input := range map[string]any{
			"embedded struct":  hidesEmbedded{inner{Big: big}},
			"embedded pointer": hidesEmbeddedPtr{&inner{Big: big}},
		} {
			obj, err := registry.FromGo(input)
			require.NoError(t, err, name)
			goStruct, ok := obj.(*object.GoStruct)
			require.True(t, ok, "%s should be wrapped as a GoStruct", name)
			_, ok = goStruct.GetAttr("Big")
			assert.False(t, ok, "%s should not expose the embedded Big field", name)
		}

		obj, err := registry.FromGo(showsEmbedded{Shown{Big: inRange}})
		require.NoError(t, err)
		goStruct, ok := obj.(*object.GoStruct)
		require.True(t, ok, "showsEmbedded should be wrapped as a GoStruct")
		_, ok = goStruct.GetAttr("Big")
		assert.False(t, ok, "Big should not be promoted from the exported embedded struct")
		shown, ok := goStruct.GetAttr("Shown")
		require.True(t, ok, "the exported embedded struct should be a field named Shown")
		shownStruct, ok := shown.(*object.GoStruct)
		require.True(t, ok, "Shown should be wrapped as a GoStruct")
		bigAttr, ok := shownStruct.GetAttr("Big")
		require.True(t, ok, "Shown.Big should be readable")
		assert.Equal(t, object.NewInt(math.MaxInt64), bigAttr)
	})

	t.Run("invalid value is skipped", func(t *testing.T) {
		assert.NotPanics(t, func() {
			assert.NoError(t, checkUintRange(reflect.Value{}, constants.Ctx))
		})
	})
}
