package internal

import (
	"net/url"
	"testing"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/stretchr/testify/require"
	starlarkLib "go.starlark.net/starlark"
)

func TestConvertStarlarkValueToInterface(t *testing.T) {
	t.Run("primitive types", func(t *testing.T) {
		tests := []struct {
			name     string
			input    starlarkLib.Value
			expected any
		}{
			{
				name:     "nil value",
				input:    nil,
				expected: nil,
			},
			{
				name:     "bool true",
				input:    starlarkLib.Bool(true),
				expected: true,
			},
			{
				name:     "bool false",
				input:    starlarkLib.Bool(false),
				expected: false,
			},
			{
				name:     "int",
				input:    starlarkLib.MakeInt(42),
				expected: int64(42),
			},
			{
				name:     "float",
				input:    starlarkLib.Float(3.14),
				expected: float64(3.14),
			},
			{
				name:     "string",
				input:    starlarkLib.String("hello"),
				expected: "hello",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := ConvertStarlarkValueToInterface(tt.input)
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("list types", func(t *testing.T) {
		tests := []struct {
			name     string
			input    *starlarkLib.List
			expected []any
		}{
			{
				name:     "empty list",
				input:    starlarkLib.NewList(nil),
				expected: []any{},
			},
			{
				name: "mixed type list",
				input: func() *starlarkLib.List {
					l := starlarkLib.NewList([]starlarkLib.Value{
						starlarkLib.MakeInt(1),
						starlarkLib.String("two"),
						starlarkLib.Bool(true),
					})
					return l
				}(),
				expected: []any{int64(1), "two", true},
			},
			{
				name: "nested list",
				input: func() *starlarkLib.List {
					inner := starlarkLib.NewList([]starlarkLib.Value{
						starlarkLib.MakeInt(1),
						starlarkLib.MakeInt(2),
					})
					outer := starlarkLib.NewList([]starlarkLib.Value{inner})
					return outer
				}(),
				expected: []any{[]any{int64(1), int64(2)}},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := ConvertStarlarkValueToInterface(tt.input)
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("dict types", func(t *testing.T) {
		tests := []struct {
			name     string
			input    *starlarkLib.Dict
			expected map[string]any
		}{
			{
				name:     "empty dict",
				input:    starlarkLib.NewDict(0),
				expected: map[string]any{},
			},
			{
				name: "string keys dict",
				input: func() *starlarkLib.Dict {
					d := starlarkLib.NewDict(1)
					require.NoError(t, d.SetKey(starlarkLib.String("key"), starlarkLib.MakeInt(42)))
					return d
				}(),
				expected: map[string]any{"key": int64(42)},
			},
			{
				name: "nested dict",
				input: func() *starlarkLib.Dict {
					inner := starlarkLib.NewDict(1)
					require.NoError(
						t,
						inner.SetKey(starlarkLib.String("inner"), starlarkLib.MakeInt(1)),
					)

					outer := starlarkLib.NewDict(1)
					require.NoError(t, outer.SetKey(starlarkLib.String("outer"), inner))
					return outer
				}(),
				expected: map[string]any{
					"outer": map[string]any{
						"inner": int64(1),
					},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := ConvertStarlarkValueToInterface(tt.input)
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name  string
			input func() *starlarkLib.Dict
		}{
			{
				name: "dict with invalid entry",
				input: func() *starlarkLib.Dict {
					d := starlarkLib.NewDict(1)
					// Create an invalid entry that will fail Get()
					err := d.Clear() // This creates an inconsistent state
					require.NoError(t, err)
					return d
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := ConvertStarlarkValueToInterface(tt.input())
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Empty(t, result.(map[string]any))
			})
		}
	})
}

func TestConvertToStarlarkFormat(t *testing.T) {
	t.Run("basic types", func(t *testing.T) {
		tests := []struct {
			name     string
			input    map[string]any
			expected starlarkLib.StringDict
			wantErr  bool
		}{
			{
				name:  "empty map",
				input: map[string]any{},
				expected: starlarkLib.StringDict{
					constants.Ctx: starlarkLib.NewDict(0),
				},
			},
			{
				name: "simple types",
				input: map[string]any{
					"bool":   true,
					"int":    42,
					"float":  3.14,
					"string": "hello",
				},
				expected: func() starlarkLib.StringDict {
					d := starlarkLib.NewDict(4)
					require.NoError(t, d.SetKey(starlarkLib.String("bool"), starlarkLib.Bool(true)))
					require.NoError(t, d.SetKey(starlarkLib.String("int"), starlarkLib.MakeInt(42)))
					require.NoError(
						t,
						d.SetKey(starlarkLib.String("float"), starlarkLib.Float(3.14)),
					)
					require.NoError(
						t,
						d.SetKey(starlarkLib.String("string"), starlarkLib.String("hello")),
					)
					return starlarkLib.StringDict{constants.Ctx: d}
				}(),
			},
			{
				name: "with nil value",
				input: map[string]any{
					"nil": nil,
				},
				expected: starlarkLib.StringDict{
					constants.Ctx: func() *starlarkLib.Dict {
						d := starlarkLib.NewDict(1)
						require.NoError(t, d.SetKey(starlarkLib.String("nil"), starlarkLib.None))
						return d
					}(),
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := ConvertToStarlarkFormat(tt.input)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Equal(t, len(tt.expected), len(result))

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
	})

	t.Run("complex types", func(t *testing.T) {
		tests := []struct {
			name     string
			input    map[string]any
			expected starlarkLib.StringDict
			wantErr  bool
		}{
			{
				name: "with URL",
				input: map[string]any{
					"url": &url.URL{Scheme: "https", Host: "localhost:8080"},
				},
				expected: func() starlarkLib.StringDict {
					d := starlarkLib.NewDict(1)
					u := &url.URL{Scheme: "https", Host: "localhost:8080"}
					require.NoError(
						t,
						d.SetKey(starlarkLib.String("url"), starlarkLib.String(u.String())),
					)
					return starlarkLib.StringDict{constants.Ctx: d}
				}(),
			},
			{
				name: "with headers",
				input: map[string]any{
					"headers": map[string][]string{
						"Accept": {"text/plain", "application/json"},
					},
				},
				expected: func() starlarkLib.StringDict {
					d := starlarkLib.NewDict(1)
					// Create inner dict for headers
					headers := starlarkLib.NewDict(1)
					// Create list for Accept values
					acceptList := starlarkLib.NewList([]starlarkLib.Value{
						starlarkLib.String("text/plain"),
						starlarkLib.String("application/json"),
					})
					// Set Accept list in headers dict
					require.NoError(t, headers.SetKey(starlarkLib.String("Accept"), acceptList))
					// Set headers dict in outer dict
					require.NoError(t, d.SetKey(starlarkLib.String("headers"), headers))
					return starlarkLib.StringDict{constants.Ctx: d}
				}(),
			},
			{
				name: "nested structures",
				input: map[string]any{
					"nested": map[string]any{
						"list": []any{1, "two", true},
					},
				},
				expected: func() starlarkLib.StringDict {
					inner := starlarkLib.NewDict(1)
					l := starlarkLib.NewList([]starlarkLib.Value{
						starlarkLib.MakeInt(1),
						starlarkLib.String("two"),
						starlarkLib.Bool(true),
					})
					require.NoError(t, inner.SetKey(starlarkLib.String("list"), l))
					d := starlarkLib.NewDict(1)
					require.NoError(t, d.SetKey(starlarkLib.String("nested"), inner))
					return starlarkLib.StringDict{constants.Ctx: d}
				}(),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := ConvertToStarlarkFormat(tt.input)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Equal(t, len(tt.expected), len(result))

				// Get ctx value and verify it's a dict
				ctxVal, ok := result[constants.Ctx].(*starlarkLib.Dict)
				require.True(t, ok, "Expected ctx value to be a dict")

				// Compare the dict contents
				expectedCtx := tt.expected[constants.Ctx].(*starlarkLib.Dict)
				require.Equal(t, expectedCtx.Len(), ctxVal.Len(), "Dict lengths should match")

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
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name    string
			input   map[string]any
			wantErr bool
		}{
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
			t.Run(tt.name, func(t *testing.T) {
				_, err := ConvertToStarlarkFormat(tt.input)
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to convert input value")
			})
		}
	})
}

func TestConvertToStarlarkValue(t *testing.T) {
	t.Run("primitive types", func(t *testing.T) {
		tests := []struct {
			name     string
			input    any
			expected starlarkLib.Value
		}{
			{
				name:     "nil",
				input:    nil,
				expected: starlarkLib.None,
			},
			{
				name:     "bool true",
				input:    true,
				expected: starlarkLib.Bool(true),
			},
			{
				name:     "bool false",
				input:    false,
				expected: starlarkLib.Bool(false),
			},
			{
				name:     "int",
				input:    42,
				expected: starlarkLib.MakeInt(42),
			},
			{
				name:     "int64",
				input:    int64(42),
				expected: starlarkLib.MakeInt64(42),
			},
			{
				name:     "float64",
				input:    3.14,
				expected: starlarkLib.Float(3.14),
			},
			{
				name:     "string",
				input:    "hello",
				expected: starlarkLib.String("hello"),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := ConvertToStarlarkValue(tt.input)
				require.NoError(t, err)
				require.Equal(t, tt.expected.String(), result.String())
				require.Equal(t, tt.expected.Type(), result.Type())
			})
		}
	})

	t.Run("URL type", func(t *testing.T) {
		tests := []struct {
			name     string
			input    *url.URL
			expected starlarkLib.Value
		}{
			{
				name: "simple URL",
				input: &url.URL{
					Scheme: "https",
					Host:   "localhost:8080",
				},
				expected: starlarkLib.String("https://localhost:8080"),
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
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := ConvertToStarlarkValue(tt.input)
				require.NoError(t, err)
				require.Equal(t, tt.expected.String(), result.String())
				require.Equal(t, tt.expected.Type(), result.Type())
			})
		}
	})

	t.Run("slice types", func(t *testing.T) {
		tests := []struct {
			name     string
			input    []any
			expected func() *starlarkLib.List
		}{
			{
				name:  "empty slice",
				input: []any{},
				expected: func() *starlarkLib.List {
					return starlarkLib.NewList([]starlarkLib.Value{})
				},
			},
			{
				name:  "mixed types",
				input: []any{42, "hello", true},
				expected: func() *starlarkLib.List {
					return starlarkLib.NewList([]starlarkLib.Value{
						starlarkLib.MakeInt(42),
						starlarkLib.String("hello"),
						starlarkLib.Bool(true),
					})
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := ConvertToStarlarkValue(tt.input)
				require.NoError(t, err)
				expected := tt.expected()
				require.Equal(t, expected.String(), result.String())
				require.Equal(t, expected.Len(), result.(*starlarkLib.List).Len())
			})
		}
	})

	t.Run("map types", func(t *testing.T) {
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
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name     string
			input    any
			errorMsg string
		}{
			{
				name:     "unsupported type",
				input:    make(chan int),
				errorMsg: "unsupported type chan int",
			},
			{
				name: "invalid nested type",
				input: []any{
					make(chan int),
				},
				errorMsg: "failed to convert list element",
			},
			{
				name: "invalid map value",
				input: map[string]any{
					"chan": make(chan int),
				},
				errorMsg: "failed to convert dict value",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := ConvertToStarlarkValue(tt.input)
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			})
		}
	})
}
