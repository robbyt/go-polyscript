// Description: This file contains constants used for accessing values from context objects.
package constants

const (
	EvalData   = "eval_data"   // object added to ctx objects sent to the evaluator, load with ctx.Value()
	Ctx        = "ctx"         // top-scope variable name for accessing input data from scripts
	Request    = "request"     // key for accessing the request object from the EvalData map
	ScriptData = "script_data" // key for accessing the "script_data" object set in the config file
)
