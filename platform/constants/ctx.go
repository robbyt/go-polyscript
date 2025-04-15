// Description: This file contains constants used for accessing values from context objects.
package constants

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// EvalData is the key used to store evaluation data in the context
	EvalData ContextKey = "eval_data" // object added to ctx objects sent to the evaluator, load with ctx.Value()

	// These are string keys used within the EvalData map, not context keys
	Ctx = "ctx" // top-scope variable name for accessing input data from scripts
)
