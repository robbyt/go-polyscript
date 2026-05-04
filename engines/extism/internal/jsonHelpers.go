package internal

import (
	"encoding/json"
	"math"
)

// convertJSONNumber converts a json.Number to:
//   - int, when the value parses as an integer and fits in the platform's int
//   - int64, when it parses as an integer but exceeds platform int (only matters on 32-bit)
//   - float64, when the value parses as a decimal or is too large to fit in int64
//   - the original json.Number unchanged, when neither parser succeeds
func convertJSONNumber(num json.Number) any {
	if n, err := num.Int64(); err == nil {
		if n >= math.MinInt && n <= math.MaxInt {
			return int(n)
		}
		return n
	}
	if n, err := num.Float64(); err == nil {
		return n
	}
	return num
}

// FixJSONNumberTypes converts json.Number values to appropriate Go types.
// Integers become int (or int64 when they exceed the platform int range),
// decimals become float64, and values that parse as neither are left as json.Number.
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
