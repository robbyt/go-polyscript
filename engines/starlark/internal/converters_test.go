package internal

import (
	"math"
	"math/big"
	"net/url"
	"testing"

	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/stretchr/testify/require"
	starlarkLib "go.starlark.net/starlark"
)

// customSequence is a non-range Indexable used to check that the range
// conversion does not swallow arbitrary sequence types.
type customSequence struct{}

func (customSequence) String() string                        { return "customSequence" }
func (customSequence) Type() string                          { return "customSequence" }
func (customSequence) Freeze()                               {}
func (customSequence) Truth() starlarkLib.Bool               { return starlarkLib.True }
func (customSequence) Hash() (uint32, error)                 { return 0, nil }
func (customSequence) Len() int                              { return 1 }
func (customSequence) Index(int) starlarkLib.Value           { return starlarkLib.MakeInt(1) }
func (customSequence) Iterate() starlarkLib.Iterator         { return nil }
func (customSequence) Slice(int, int, int) starlarkLib.Value { return starlarkLib.None }

// starlarkToGoCase is one ConvertStarlarkValueToInterface table entry.
type starlarkToGoCase struct {
	name     string
	input    starlarkLib.Value
	expected any
	wantErr  bool
}

// makeRange calls the builtin range(), since the range type is unexported.
func makeRange(t *testing.T, args ...int) starlarkLib.Value {
	t.Helper()
	tuple := make(starlarkLib.Tuple, len(args))
	for i, a := range args {
		tuple[i] = starlarkLib.MakeInt(a)
	}
	v, err := starlarkLib.Call(&starlarkLib.Thread{}, starlarkLib.Universe["range"], tuple, nil)
	require.NoError(t, err)
	return v
}

// makeSet builds a set holding vals in insertion order.
func makeSet(t *testing.T, vals ...starlarkLib.Value) *starlarkLib.Set {
	t.Helper()
	s := starlarkLib.NewSet(len(vals))
	for _, v := range vals {
		require.NoError(t, s.Insert(v), "inserting into set should succeed")
	}
	return s
}

// makeStringDict builds a dict with a single string key.
func makeStringDict(t *testing.T, key string, value starlarkLib.Value) *starlarkLib.Dict {
	t.Helper()
	d := starlarkLib.NewDict(1)
	require.NoError(t, d.SetKey(starlarkLib.String(key), value), "setting dict key should succeed")
	return d
}

// bigPow2 returns 2^n as a *big.Int.
func bigPow2(n uint) *big.Int {
	return new(big.Int).Lsh(big.NewInt(1), n)
}

func starlarkToGoPrimitiveCases() []starlarkToGoCase {
	return []starlarkToGoCase{
		{name: "nil value", input: nil, expected: nil},
		{name: "bool true", input: starlarkLib.Bool(true), expected: true},
		{name: "bool false", input: starlarkLib.Bool(false), expected: false},
		{name: "int", input: starlarkLib.MakeInt(42), expected: int64(42)},
		{
			name:     "int at int64 max",
			input:    starlarkLib.MakeInt64(math.MaxInt64),
			expected: int64(math.MaxInt64),
		},
		{
			name:     "int at int64 min",
			input:    starlarkLib.MakeInt64(math.MinInt64),
			expected: int64(math.MinInt64),
		},
		{
			name:     "int above int64 max",
			input:    starlarkLib.MakeBigInt(bigPow2(64)),
			expected: bigPow2(64),
		},
		{
			name:     "int below int64 min",
			input:    starlarkLib.MakeBigInt(new(big.Int).Neg(bigPow2(70))),
			expected: new(big.Int).Neg(bigPow2(70)),
		},
		{name: "float", input: starlarkLib.Float(3.14), expected: float64(3.14)},
		{name: "string", input: starlarkLib.String("hello"), expected: "hello"},
		{name: "bytes", input: starlarkLib.Bytes("abc"), expected: []byte("abc")},
	}
}

func starlarkToGoSequenceCases(t *testing.T) []starlarkToGoCase {
	t.Helper()
	return []starlarkToGoCase{
		{name: "empty list", input: starlarkLib.NewList(nil), expected: []any{}},
		{
			name: "mixed type list",
			input: starlarkLib.NewList([]starlarkLib.Value{
				starlarkLib.MakeInt(1),
				starlarkLib.String("two"),
				starlarkLib.Bool(true),
			}),
			expected: []any{int64(1), "two", true},
		},
		{
			name: "nested list",
			input: starlarkLib.NewList([]starlarkLib.Value{
				starlarkLib.NewList([]starlarkLib.Value{starlarkLib.MakeInt(1), starlarkLib.MakeInt(2)}),
			}),
			expected: []any{[]any{int64(1), int64(2)}},
		},
		{name: "empty tuple", input: starlarkLib.Tuple{}, expected: []any{}},
		{
			name: "nested tuple",
			input: starlarkLib.Tuple{
				starlarkLib.MakeInt(1),
				starlarkLib.Tuple{starlarkLib.String("a"), starlarkLib.None},
			},
			expected: []any{int64(1), []any{"a", nil}},
		},
		{
			name: "set keeps insertion order",
			input: makeSet(t,
				starlarkLib.MakeInt(3),
				starlarkLib.String("x"),
				starlarkLib.Tuple{starlarkLib.MakeInt(1)},
			),
			expected: []any{int64(3), "x", []any{int64(1)}},
		},
		{name: "empty set", input: starlarkLib.NewSet(0), expected: []any{}},
		{
			name: "list containing big int and bytes",
			input: starlarkLib.NewList([]starlarkLib.Value{
				starlarkLib.MakeBigInt(bigPow2(64)),
				starlarkLib.Bytes("z"),
			}),
			expected: []any{bigPow2(64), []byte("z")},
		},
	}
}

func starlarkToGoRangeCases(t *testing.T) []starlarkToGoCase {
	t.Helper()
	return []starlarkToGoCase{
		{name: "range", input: makeRange(t, 3), expected: []any{int64(0), int64(1), int64(2)}},
		{
			name:     "stepped range",
			input:    makeRange(t, 10, 0, -4),
			expected: []any{int64(10), int64(6), int64(2)},
		},
		{name: "empty range", input: makeRange(t, 0), expected: []any{}},
		{
			name:     "range nested in list",
			input:    starlarkLib.NewList([]starlarkLib.Value{makeRange(t, 2)}),
			expected: []any{[]any{int64(0), int64(1)}},
		},
		{name: "non-range indexable", input: customSequence{}, wantErr: true},
	}
}

func starlarkToGoDictCases(t *testing.T) []starlarkToGoCase {
	t.Helper()

	// A cleared dict should convert to an empty map without an error.
	cleared := makeStringDict(t, "key", starlarkLib.MakeInt(1))
	require.NoError(t, cleared.Clear(), "clearing dict should succeed")

	return []starlarkToGoCase{
		{name: "empty dict", input: starlarkLib.NewDict(0), expected: map[string]any{}},
		{
			name:     "string keys dict",
			input:    makeStringDict(t, "key", starlarkLib.MakeInt(42)),
			expected: map[string]any{"key": int64(42)},
		},
		{
			name:     "nested dict",
			input:    makeStringDict(t, "outer", makeStringDict(t, "inner", starlarkLib.MakeInt(1))),
			expected: map[string]any{"outer": map[string]any{"inner": int64(1)}},
		},
		{name: "cleared dict", input: cleared, expected: map[string]any{}},
	}
}

// runStarlarkToGoCases runs each case through ConvertStarlarkValueToInterface.
func runStarlarkToGoCases(t *testing.T, tests []starlarkToGoCase) {
	t.Helper()
	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertStarlarkValueToInterface(tt.input)
			if tt.wantErr {
				require.Error(t, err, "conversion should fail")
				return
			}
			require.NoError(t, err, "conversion should succeed")
			require.Equal(t, tt.expected, result, "converted value should match")
		})
	}
}

func TestConvertStarlarkValueToInterface(t *testing.T) {
	t.Parallel()

	t.Run("primitives", func(t *testing.T) {
		runStarlarkToGoCases(t, starlarkToGoPrimitiveCases())
	})
	t.Run("sequences", func(t *testing.T) {
		runStarlarkToGoCases(t, starlarkToGoSequenceCases(t))
	})
	t.Run("ranges", func(t *testing.T) {
		runStarlarkToGoCases(t, starlarkToGoRangeCases(t))
	})
	t.Run("dicts", func(t *testing.T) {
		runStarlarkToGoCases(t, starlarkToGoDictCases(t))
	})
}

func TestConvertToStarlarkFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    map[string]any
		expected starlarkLib.StringDict
		wantErr  bool
	}{
		// Basic types
		{
			name:  "empty map",
			input: map[string]any{},
			expected: starlarkLib.StringDict{
				constants.Ctx: starlarkLib.NewDict(0),
			},
			wantErr: false,
		},
		{
			name: "simple types",
			input: map[string]any{
				"bool":   true,
				"int":    42,
				"float":  3.14,
				"string": "hello",
			},
			expected: starlarkLib.StringDict{
				constants.Ctx: func() *starlarkLib.Dict {
					d := starlarkLib.NewDict(4)
					if err := d.SetKey(starlarkLib.String("bool"), starlarkLib.Bool(true)); err != nil {
						t.Fatalf("Failed to set key: %v", err)
					}
					if err := d.SetKey(starlarkLib.String("int"), starlarkLib.MakeInt(42)); err != nil {
						t.Fatalf("Failed to set key: %v", err)
					}
					if err := d.SetKey(starlarkLib.String("float"), starlarkLib.Float(3.14)); err != nil {
						t.Fatalf("Failed to set key: %v", err)
					}
					if err := d.SetKey(starlarkLib.String("string"), starlarkLib.String("hello")); err != nil {
						t.Fatalf("Failed to set key: %v", err)
					}
					return d
				}(),
			},
			wantErr: false,
		},
		{
			name: "with nil value",
			input: map[string]any{
				"nil": nil,
			},
			expected: starlarkLib.StringDict{
				constants.Ctx: func() *starlarkLib.Dict {
					d := starlarkLib.NewDict(1)
					if err := d.SetKey(starlarkLib.String("nil"), starlarkLib.None); err != nil {
						t.Fatalf("Failed to set nil key: %v", err)
					}
					return d
				}(),
			},
			wantErr: false,
		},

		// Complex types
		{
			name: "with URL",
			input: map[string]any{
				"url": &url.URL{Scheme: "https", Host: "localhost:8080"},
			},
			expected: starlarkLib.StringDict{
				constants.Ctx: func() *starlarkLib.Dict {
					d := starlarkLib.NewDict(1)
					u := &url.URL{Scheme: "https", Host: "localhost:8080"}
					if err := d.SetKey(starlarkLib.String("url"), starlarkLib.String(u.String())); err != nil {
						t.Fatalf("Failed to set url key: %v", err)
					}
					return d
				}(),
			},
			wantErr: false,
		},
		{
			name: "with headers",
			input: map[string]any{
				"headers": map[string][]string{
					"Accept": {"text/plain", "application/json"},
				},
			},
			expected: starlarkLib.StringDict{
				constants.Ctx: func() *starlarkLib.Dict {
					d := starlarkLib.NewDict(1)
					headers := starlarkLib.NewDict(1)
					acceptList := starlarkLib.NewList([]starlarkLib.Value{
						starlarkLib.String("text/plain"),
						starlarkLib.String("application/json"),
					})
					if err := headers.SetKey(starlarkLib.String("Accept"), acceptList); err != nil {
						t.Fatalf("Failed to set Accept key: %v", err)
					}
					if err := d.SetKey(starlarkLib.String("headers"), headers); err != nil {
						t.Fatalf("Failed to set headers key: %v", err)
					}
					return d
				}(),
			},
			wantErr: false,
		},
		{
			name: "nested structures",
			input: map[string]any{
				"nested": map[string]any{
					"list": []any{1, "two", true},
				},
			},
			expected: starlarkLib.StringDict{
				constants.Ctx: func() *starlarkLib.Dict {
					inner := starlarkLib.NewDict(1)
					l := starlarkLib.NewList([]starlarkLib.Value{
						starlarkLib.MakeInt(1),
						starlarkLib.String("two"),
						starlarkLib.Bool(true),
					})
					if err := inner.SetKey(starlarkLib.String("list"), l); err != nil {
						t.Fatalf("Failed to set list key: %v", err)
					}
					d := starlarkLib.NewDict(1)
					if err := d.SetKey(starlarkLib.String("nested"), inner); err != nil {
						t.Fatalf("Failed to set nested key: %v", err)
					}
					return d
				}(),
			},
			wantErr: false,
		},

		// Error cases
		{
			name: "unsupported type",
			input: map[string]any{
				"chan": make(chan int),
			},
			wantErr: true,
		},
		{
			name: "mixed valid and invalid",
			input: map[string]any{
				"valid":   "value",
				"invalid": make(chan int),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertToStarlarkFormat(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				// Without defined error sentinel values, we can directly check the message
				// In a more ideal harmonization, we would define and use error sentinels
				require.Contains(t, err.Error(), "failed to convert input value")
				return
			}

			require.NoError(t, err)
			require.Len(t, result, len(tt.expected))

			// Get ctx value and verify it's a dict
			ctxVal, ok := result[constants.Ctx].(*starlarkLib.Dict)
			require.True(t, ok)

			// Compare the dict contents
			expectedCtx := tt.expected[constants.Ctx].(*starlarkLib.Dict)
			require.Equal(t, expectedCtx.Len(), ctxVal.Len())

			for _, k := range expectedCtx.Keys() {
				expectedVal, found, err := expectedCtx.Get(k)
				require.NoError(t, err)
				require.True(t, found)

				actualVal, found, err := ctxVal.Get(k)
				require.NoError(t, err)
				require.True(t, found)

				require.Equal(t, expectedVal.String(), actualVal.String())
			}
		})
	}
}

func TestConvertToStarlarkValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected starlarkLib.Value
		checkFn  func(t *testing.T, expected, actual starlarkLib.Value)
		wantErr  bool
		errMsg   string
	}{
		// Primitive types
		{
			name:     "nil",
			input:    nil,
			expected: starlarkLib.None,
			checkFn: func(t *testing.T, expected, actual starlarkLib.Value) {
				t.Helper()
				require.Equal(t, expected.String(), actual.String())
				require.Equal(t, expected.Type(), actual.Type())
			},
		},
		{
			name:     "bool true",
			input:    true,
			expected: starlarkLib.Bool(true),
			checkFn: func(t *testing.T, expected, actual starlarkLib.Value) {
				t.Helper()
				require.Equal(t, expected.String(), actual.String())
				require.Equal(t, expected.Type(), actual.Type())
			},
		},
		{
			name:     "bool false",
			input:    false,
			expected: starlarkLib.Bool(false),
			checkFn: func(t *testing.T, expected, actual starlarkLib.Value) {
				t.Helper()
				require.Equal(t, expected.String(), actual.String())
				require.Equal(t, expected.Type(), actual.Type())
			},
		},
		{
			name:     "int",
			input:    42,
			expected: starlarkLib.MakeInt(42),
			checkFn: func(t *testing.T, expected, actual starlarkLib.Value) {
				t.Helper()
				require.Equal(t, expected.String(), actual.String())
				require.Equal(t, expected.Type(), actual.Type())
			},
		},
		{
			name:     "int64",
			input:    int64(42),
			expected: starlarkLib.MakeInt64(42),
			checkFn: func(t *testing.T, expected, actual starlarkLib.Value) {
				t.Helper()
				require.Equal(t, expected.String(), actual.String())
				require.Equal(t, expected.Type(), actual.Type())
			},
		},
		{
			name:     "float64",
			input:    3.14,
			expected: starlarkLib.Float(3.14),
			checkFn: func(t *testing.T, expected, actual starlarkLib.Value) {
				t.Helper()
				require.Equal(t, expected.String(), actual.String())
				require.Equal(t, expected.Type(), actual.Type())
			},
		},
		{
			name:     "string",
			input:    "hello",
			expected: starlarkLib.String("hello"),
			checkFn: func(t *testing.T, expected, actual starlarkLib.Value) {
				t.Helper()
				require.Equal(t, expected.String(), actual.String())
				require.Equal(t, expected.Type(), actual.Type())
			},
		},

		// URL types
		{
			name: "simple URL",
			input: &url.URL{
				Scheme: "https",
				Host:   "localhost:8080",
			},
			expected: starlarkLib.String("https://localhost:8080"),
			checkFn: func(t *testing.T, expected, actual starlarkLib.Value) {
				t.Helper()
				require.Equal(t, expected.String(), actual.String())
				require.Equal(t, expected.Type(), actual.Type())
			},
		},
		{
			name: "complex URL",
			input: &url.URL{
				Scheme:   "https",
				Host:     "localhost:8080",
				Path:     "/path",
				RawQuery: "q=search",
			},
			expected: starlarkLib.String("https://localhost:8080/path?q=search"),
			checkFn: func(t *testing.T, expected, actual starlarkLib.Value) {
				t.Helper()
				require.Equal(t, expected.String(), actual.String())
				require.Equal(t, expected.Type(), actual.Type())
			},
		},

		// Slice types
		{
			name:     "empty slice",
			input:    []any{},
			expected: starlarkLib.NewList([]starlarkLib.Value{}),
			checkFn: func(t *testing.T, expected, actual starlarkLib.Value) {
				t.Helper()
				require.Equal(t, expected.String(), actual.String())
				require.Equal(
					t,
					expected.(*starlarkLib.List).Len(),
					actual.(*starlarkLib.List).Len(),
				)
			},
		},
		{
			name:  "mixed type slice",
			input: []any{42, "hello", true},
			expected: starlarkLib.NewList([]starlarkLib.Value{
				starlarkLib.MakeInt(42),
				starlarkLib.String("hello"),
				starlarkLib.Bool(true),
			}),
			checkFn: func(t *testing.T, expected, actual starlarkLib.Value) {
				t.Helper()
				require.Equal(t, expected.String(), actual.String())
				require.Equal(
					t,
					expected.(*starlarkLib.List).Len(),
					actual.(*starlarkLib.List).Len(),
				)
			},
		},

		// Map types - We'll test these separately as they need special verification

		// Error cases
		{
			name:    "unsupported type",
			input:   make(chan int),
			wantErr: true,
			errMsg:  "unsupported type chan int",
		},
		{
			name: "invalid nested type",
			input: []any{
				make(chan int),
			},
			wantErr: true,
			errMsg:  "failed to convert list element",
		},
		{
			name: "invalid map value",
			input: map[string]any{
				"chan": make(chan int),
			},
			wantErr: true,
			errMsg:  "failed to convert dict value",
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertToStarlarkValue(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			if tt.checkFn != nil {
				tt.checkFn(t, tt.expected, result)
			}
		})
	}

	// Test map types separately as they need more detailed verification
	t.Run("map[string][]string type", func(t *testing.T) {
		input := map[string][]string{
			"headers": {"value1", "value2"},
			"empty":   {},
		}
		result, err := ConvertToStarlarkValue(input)
		require.NoError(t, err)
		dict := result.(*starlarkLib.Dict)

		// Check headers
		headersVal, found, err := dict.Get(starlarkLib.String("headers"))
		require.NoError(t, err)
		require.True(t, found)
		headersList := headersVal.(*starlarkLib.List)
		require.Equal(t, 2, headersList.Len())
		val1 := starlarkLib.String("value1")
		val2 := starlarkLib.String("value2")
		require.Equal(t, val1, headersList.Index(0))
		require.Equal(t, val2, headersList.Index(1))

		// Check empty
		emptyVal, found, err := dict.Get(starlarkLib.String("empty"))
		require.NoError(t, err)
		require.True(t, found)
		emptyList := emptyVal.(*starlarkLib.List)
		require.Equal(t, 0, emptyList.Len())
	})

	t.Run("map[string]any type", func(t *testing.T) {
		input := map[string]any{
			"int":    42,
			"str":    "hello",
			"nested": map[string]any{"key": "value"},
		}

		result, err := ConvertToStarlarkValue(input)
		require.NoError(t, err)
		dict := result.(*starlarkLib.Dict)

		// Check int value
		intVal, found, err := dict.Get(starlarkLib.String("int"))
		require.NoError(t, err)
		require.True(t, found)
		expectedInt := starlarkLib.MakeInt(42)
		require.Equal(t, expectedInt, intVal)

		// Check string value
		strVal, found, err := dict.Get(starlarkLib.String("str"))
		require.NoError(t, err)
		require.True(t, found)
		expectedStr := starlarkLib.String("hello")
		require.Equal(t, expectedStr, strVal)

		// Check nested dict
		nestedVal, found, err := dict.Get(starlarkLib.String("nested"))
		require.NoError(t, err)
		require.True(t, found)
		nestedDict := nestedVal.(*starlarkLib.Dict)

		keyVal, found, err := nestedDict.Get(starlarkLib.String("key"))
		require.NoError(t, err)
		require.True(t, found)
		expectedKeyVal := starlarkLib.String("value")
		require.Equal(t, expectedKeyVal, keyVal)
	})
}
