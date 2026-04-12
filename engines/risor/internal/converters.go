package internal

import (
	risor "github.com/deepnoodle-ai/risor/v2"
)

// BuildRisorEnv builds the full Risor environment map with standard builtins and input data.
// The input data is made available under the given ctxKey (typically "ctx").
//
// For example, if the inputData is {"foo": "bar", "baz": 123}, the output will be a map
// containing all standard Risor builtins plus:
//
//	"ctx": map[string]any{
//	    "foo": "bar",
//	    "baz": 123,
//	}
func BuildRisorEnv(ctxKey string, inputData map[string]any) map[string]any {
	// Builtins() returns a fresh map on each call, so mutating env is safe.
	env := risor.Builtins()
	env[ctxKey] = inputData
	return env
}
