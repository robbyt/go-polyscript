package internal

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/robbyt/go-polyscript/platform/constants"
	starlarkLib "go.starlark.net/starlark"
)

// ConvertToStarlarkFormat converts a Go map into Starlark StringDict format.
// It wraps the input data in a "ctx" object that will be accessible within the script.
func ConvertToStarlarkFormat(inputData map[string]any) (starlarkLib.StringDict, error) {
	// Start with the ctx dict
	sDict := make(starlarkLib.StringDict, 1)

	// Create a Starlark dict for the ctx global variable
	ctxDict := starlarkLib.NewDict(len(inputData))

	// Convert each input data key-value pair and add to the ctxDict
	errz := make([]error, 0, len(inputData))
	for k, v := range inputData {
		starlarkVal, err := ConvertToStarlarkValue(v)
		if err != nil {
			// Collect errors but continue processing
			errz = append(errz, fmt.Errorf("failed to convert input value for key %q: %w", k, err))
			continue
		}
		if err := ctxDict.SetKey(starlarkLib.String(k), starlarkVal); err != nil {
			// Collect errors but continue processing
			errz = append(errz, fmt.Errorf("failed to set ctx dict key %q: %w", k, err))
			continue
		}
	}

	// return if there were any errors
	if len(errz) > 0 {
		return nil, errors.Join(errz...)
	}

	// Set the ctx global variable to the dictionary
	sDict[constants.Ctx] = ctxDict
	return sDict, nil
}

// ConvertStarlarkValueToInterface converts a Starlark value to a Go any value
func ConvertStarlarkValueToInterface(v starlarkLib.Value) (any, error) {
	if v == nil {
		return nil, nil
	}

	switch v := v.(type) {
	case starlarkLib.NoneType:
		// Return nil for None values
		return nil, nil
	case starlarkLib.Bool:
		return bool(v), nil
	case starlarkLib.Int:
		i, _ := v.Int64()
		return i, nil
	case starlarkLib.Float:
		return float64(v), nil
	case starlarkLib.String:
		return string(v), nil
	case *starlarkLib.List:
		list := make([]any, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem, err := ConvertStarlarkValueToInterface(v.Index(i))
			if err != nil {
				return nil, fmt.Errorf("failed to convert list element: %w", err)
			}
			list = append(list, elem)
		}
		return list, nil
	case *starlarkLib.Dict:
		// Create a string-keyed map for JSON compatibility
		dict := make(map[string]any)
		for _, k := range v.Keys() {
			val, found, err := v.Get(k)
			if err != nil || !found {
				continue // Skip invalid entries
			}

			// For JSON compatibility, we need string keys
			kStr, ok := k.(starlarkLib.String)
			if !ok {
				// Convert non-string keys to strings for JSON compatibility
				kStr = starlarkLib.String(k.String())
			}

			vv, err := ConvertStarlarkValueToInterface(val)
			if err != nil {
				return nil, fmt.Errorf("failed to convert dict value: %w", err)
			}
			dict[string(kStr)] = vv
		}
		return dict, nil
	// NoneType is already handled in the first case
	default:
		return nil, fmt.Errorf("unsupported Starlark type %T", v)
	}
}

// ConvertToStarlarkValue converts a Go value to a Starlark value
func ConvertToStarlarkValue(v any) (starlarkLib.Value, error) {
	if v == nil {
		return starlarkLib.None, nil
	}

	switch val := v.(type) {
	case bool:
		return starlarkLib.Bool(val), nil
	case int:
		return starlarkLib.MakeInt(val), nil
	case int64:
		return starlarkLib.MakeInt64(val), nil
	case float64:
		return starlarkLib.Float(val), nil
	case string:
		return starlarkLib.String(val), nil
	case *url.URL:
		return starlarkLib.String(val.String()), nil
	case []any:
		elements := make([]starlarkLib.Value, len(val))
		for i, elem := range val {
			var err error
			elements[i], err = ConvertToStarlarkValue(elem)
			if err != nil {
				return nil, fmt.Errorf("failed to convert list element: %w", err)
			}
		}
		return starlarkLib.NewList(elements), nil
	case map[string]struct{}:
		// golang doesn't have a Set, but often a map[string]struct{} is used instead
		elements := starlarkLib.NewSet(len(val))
		for k := range val {
			if err := elements.Insert(starlarkLib.String(k)); err != nil {
				return nil, fmt.Errorf("failed to insert set element: %w", err)
			}
		}
		return elements, nil
	case map[string][]string:
		// Special handling for HTTP headers and query params
		dict := starlarkLib.NewDict(len(val))
		for k, values := range val {
			// Convert string slice to Starlark list
			elements := make([]starlarkLib.Value, len(values))
			for i, v := range values {
				elements[i] = starlarkLib.String(v)
			}
			if err := dict.SetKey(starlarkLib.String(k), starlarkLib.NewList(elements)); err != nil {
				return nil, fmt.Errorf("failed to set dict key: %w", err)
			}
		}
		return dict, nil
	case map[string]any:
		dict := starlarkLib.NewDict(len(val))
		for k, v := range val {
			starlarkVal, err := ConvertToStarlarkValue(v)
			if err != nil {
				return nil, fmt.Errorf("failed to convert dict value: %w", err)
			}
			if err := dict.SetKey(starlarkLib.String(k), starlarkVal); err != nil {
				return nil, fmt.Errorf("failed to set dict key: %w", err)
			}
		}
		return dict, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}
