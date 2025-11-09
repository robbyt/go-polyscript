package internal

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToExtismFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    map[string]any
		wantJSON bool
		wantErr  bool
	}{
		{
			name:     "empty map",
			input:    map[string]any{},
			wantJSON: false,
			wantErr:  false,
		},
		{
			name: "valid data",
			input: map[string]any{
				"str":  "value",
				"int":  42,
				"bool": true,
				"map": map[string]any{
					"nested": "data",
				},
			},
			wantJSON: true,
			wantErr:  false,
		},
		{
			name:     "nil map",
			input:    nil,
			wantJSON: false,
			wantErr:  false,
		},
		{
			name: "complex nested data",
			input: map[string]any{
				"array": []int{1, 2, 3},
				"nested": map[string]any{
					"deeper": map[string]any{
						"evenDeeper": []map[string]any{
							{"key": "value"},
						},
					},
				},
			},
			wantJSON: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertToExtismFormat(tt.input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.wantJSON {
					require.NotNil(t, result)
					require.NotEmpty(t, result)

					// Verify the JSON can be unmarshaled
					var checkData map[string]any
					unmarshalErr := json.Unmarshal(result, &checkData)
					require.NoError(t, unmarshalErr)

					// For simple maps, verify the data matches
					if len(tt.input) > 0 && len(tt.input) < 5 {
						for k, expectedVal := range tt.input {
							// Skip complex nested structures in this check
							if _, isMap := expectedVal.(map[string]any); !isMap {
								// Handle type conversions that happen during JSON marshaling
								if intVal, isInt := expectedVal.(int); isInt {
									// JSON unmarshaling converts numbers to float64
									assert.InDelta(t, float64(intVal), checkData[k], 0.0001)
								} else if _, isIntSlice := expectedVal.([]int); isIntSlice {
									// Skip int slice checks (arrays become []any)
								} else if _, isSlice := expectedVal.([]any); !isSlice {
									assert.Equal(t, expectedVal, checkData[k])
								}
							}
						}
					}
				} else if len(tt.input) == 0 || tt.input == nil {
					require.Nil(t, result)
				}
			}
		})
	}
}
