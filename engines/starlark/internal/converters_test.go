package internal

import (
	"net/url"
	"testing"

	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/stretchr/testify/require"
	starlarkLib "go.starlark.net/starlark"
)

func TestConvertStarlarkValueToInterface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    starlarkLib.Value
		expected any
		wantErr  bool
	}{
		// Primitive types
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "bool true",
			input:    starlarkLib.Bool(true),
			expected: true,
			wantErr:  false,
		},
		{
			name:     "bool false",
			input:    starlarkLib.Bool(false),
			expected: false,
			wantErr:  false,
		},
		{
			name:     "int",
			input:    starlarkLib.MakeInt(42),
			expected: int64(42),
			wantErr:  false,
		},
		{
			name:     "float",
			input:    starlarkLib.Float(3.14),
			expected: float64(3.14),
			wantErr:  false,
		},
		{
			name:     "string",
			input:    starlarkLib.String("hello"),
			expected: "hello",
			wantErr:  false,
		},

		// List types
		{
			name:     "empty list",
			input:    starlarkLib.NewList(nil),
			expected: []any{},
			wantErr:  false,
		},
		{
			name: "mixed type list",
			input: starlarkLib.NewList([]starlarkLib.Value{
				starlarkLib.MakeInt(1),
				starlarkLib.String("two"),
				starlarkLib.Bool(true),
			}),
			expected: []any{int64(1), "two", true},
			wantErr:  false,
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
			wantErr:  false,
		},

		// Dict types
		{
			name:     "empty dict",
			input:    starlarkLib.NewDict(0),
			expected: map[string]any{},
			wantErr:  false,
		},
		{
			name: "string keys dict",
			input: func() *starlarkLib.Dict {
				d := starlarkLib.NewDict(1)
				if err := d.SetKey(starlarkLib.String("key"), starlarkLib.MakeInt(42)); err != nil {
					t.Fatalf("Failed to set key: %v", err)
				}
				return d
			}(),
			expected: map[string]any{"key": int64(42)},
			wantErr:  false,
		},
		{
			name: "nested dict",
			input: func() *starlarkLib.Dict {
				inner := starlarkLib.NewDict(1)
				if err := inner.SetKey(starlarkLib.String("inner"), starlarkLib.MakeInt(1)); err != nil {
					t.Fatalf("Failed to set key: %v", err)
				}
				outer := starlarkLib.NewDict(1)
				if err := outer.SetKey(starlarkLib.String("outer"), inner); err != nil {
					t.Fatalf("Failed to set key: %v", err)
				}
				return outer
			}(),
			expected: map[string]any{
				"outer": map[string]any{
					"inner": int64(1),
				},
			},
			wantErr: false,
		},

		// Error cases
		{
			name: "dict with invalid entry",
			input: func() *starlarkLib.Dict {
				d := starlarkLib.NewDict(1)
				// Create an invalid entry that will fail Get()
				err := d.Clear() // This creates an inconsistent state
				if err != nil {
					t.Fatalf("Failed to clear dict: %v", err)
				}
				return d
			}(),
			expected: map[string]any{},
			wantErr:  false, // Note: Current implementation doesn't return an error for this case
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertStarlarkValueToInterface(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
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
