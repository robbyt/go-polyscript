package internal

import (
	risor "github.com/deepnoodle-ai/risor/v2"
)

// BuildRisorEnv builds the full Risor environment map with standard builtins and input data.
// The input data is made available under the given ctxKey (typically "ctx").
//
// For example, with ctxKey "ctx" and inputData {"foo": "bar", "baz": 123}, the
// returned map holds every standard Risor builtin plus a "ctx" entry whose value
// is inputData itself, so a script reads ctx["foo"] as "bar" and ctx["baz"] as 123.
func BuildRisorEnv(ctxKey string, inputData map[string]any) map[string]any {
	// Builtins() returns a fresh map on each call, so mutating env is safe.
	env := risor.Builtins()
	env[ctxKey] = inputData
	return env
}
