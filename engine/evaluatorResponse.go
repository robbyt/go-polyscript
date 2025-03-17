package engine

import "github.com/robbyt/go-polyscript/execution/data"

// EvaluatorResponse is based on the risor object.Object interface, but with some features removed
type EvaluatorResponse interface {
	// Type of the object.
	Type() data.Types

	// Inspect returns a string representation of the given object.
	Inspect() string

	// Interface converts the given object to a native Go value.
	Interface() any

	// GetScriptExeID returns the ID of the script that generated the object.
	GetScriptExeID() string

	// GetExecTime returns the time it took to execute the script
	GetExecTime() string
}
