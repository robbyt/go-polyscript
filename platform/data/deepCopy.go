package data

// deepCopyMap returns a recursive copy of m. Nested map[string]any and
// []any values are allocated fresh at every level (see deepCopyValue), so
// the returned map shares no mutable container storage with m.
//
// Always returns a non-nil map, even when m is nil or empty; this lets
// callers use the result directly without a separate nil guard.
//
// Providers use it on both sides of their storage: when storing data
// (NewStaticProvider, ContextProvider.AddDataToContext deriving from a
// parent context) and when handing it out (GetData), so neither callers
// nor script engines can mutate a provider's data in place.
func deepCopyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = deepCopyValue(v)
	}
	return out
}

// deepCopyValue returns a copy of v that shares no map[string]any or []any
// storage with it. These are the container shapes the providers and the
// script engines' Go conversions use for structured data, so they are
// copied recursively, including maps nested inside slices. A nil map or
// slice stays nil.
//
// Every other value is returned as is. Value types (int, string, struct,
// etc.) are copied by the assignment; other reference types (typed slices
// such as []string, maps other than map[string]any, pointers, channels,
// function values) remain shared with the source and must not be mutated
// in place by whoever holds them.
func deepCopyValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		if val == nil {
			return val
		}
		return deepCopyMap(val)
	case []any:
		if val == nil {
			return val
		}
		out := make([]any, len(val))
		for i, elem := range val {
			out[i] = deepCopyValue(elem)
		}
		return out
	default:
		return v
	}
}
