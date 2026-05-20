package platform

import (
	"time"

	"github.com/robbyt/go-polyscript/platform/data"
)

// EvaluatorResponse is based on the risor object.Object interface, but with some features removed
type EvaluatorResponse interface {
	// Type of the object.
	Type() data.Types

	// Inspect returns a string representation of the given object.
	Inspect() string

	// Interface converts the given object to a native Go value.
	Interface() any

	// ScriptExeID returns the ID of the script that generated the object.
	ScriptExeID() string

	// ExecTime returns the time it took to execute the script.
	ExecTime() time.Duration

	// AsMap returns the result as a map[string]any. Returns an error of the
	// form "AsMap: expected map[string]any, got <T>" when the underlying value
	// has a different shape.
	AsMap() (map[string]any, error)
}
