package extism

import (
	"encoding/json"
	"strings"
)

// fixJSONNumberTypes converts json.Number values to appropriate Go types based on semantic rules
func fixJSONNumberTypes(data any) any {
	switch v := data.(type) {
	case map[string]any:
		// Process each key in the map
		for k, val := range v {
			// Handle nested structures recursively
			if nestedMap, ok := val.(map[string]any); ok {
				v[k] = fixJSONNumberTypes(nestedMap)
				continue
			}

			if nestedSlice, ok := val.([]any); ok {
				v[k] = fixJSONNumberTypes(nestedSlice)
				continue
			}

			// Convert json.Number to appropriate type
			if num, ok := val.(json.Number); ok {
				// Fields that should be integers
				if strings.HasSuffix(k, "_count") || k == "count" ||
					strings.HasSuffix(k, "_id") || strings.HasSuffix(k, "Id") {
					if n, err := num.Int64(); err == nil {
						v[k] = int(n)
					}
					continue
				}

				// Default to float64 for other numeric fields
				if n, err := num.Float64(); err == nil {
					v[k] = n
				}
			}
		}
		return v

	case []any:
		// Process each item in the slice
		for i, item := range v {
			v[i] = fixJSONNumberTypes(item)
		}
		return v

	default:
		return data
	}
}

func marshalInputData(inputData map[string]any) ([]byte, error) {
	if len(inputData) == 0 {
		return nil, nil
	}
	return json.Marshal(inputData)
}
