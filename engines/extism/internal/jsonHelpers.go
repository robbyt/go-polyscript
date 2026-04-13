package internal

import (
	"encoding/json"
)

// convertJSONNumber converts a json.Number to an int (if it fits) or float64.
func convertJSONNumber(num json.Number) any {
	if n, err := num.Int64(); err == nil {
		return int(n)
	}
	if n, err := num.Float64(); err == nil {
		return n
	}
	return num
}

// FixJSONNumberTypes converts json.Number values to appropriate Go types.
// Integer-representable numbers become int; all others become float64.
func FixJSONNumberTypes(data any) any {
	switch v := data.(type) {
	case map[string]any:
		for k, val := range v {
			if num, ok := val.(json.Number); ok {
				v[k] = convertJSONNumber(num)
				continue
			}
			v[k] = FixJSONNumberTypes(val)
		}
		return v

	case []any:
		for i, item := range v {
			if num, ok := item.(json.Number); ok {
				v[i] = convertJSONNumber(num)
				continue
			}
			v[i] = FixJSONNumberTypes(item)
		}
		return v

	default:
		return data
	}
}
