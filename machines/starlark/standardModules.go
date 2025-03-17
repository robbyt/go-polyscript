package starlark

import (
	"maps"

	starlarkJSON "go.starlark.net/lib/json"
	starlarkMath "go.starlark.net/lib/math"
	starlarkTime "go.starlark.net/lib/time"
	starlarkLib "go.starlark.net/starlark"
)

// Module namespace constants used in both compilation and execution phases
// These must be defined identically in both places to ensure scripts compile and run correctly
const (
	namespaceJSON = "json" // Provides JSON encoding/decoding functions
	namespaceMath = "math" // Provides mathematical functions and constants
	namespaceTime = "time" // Provides time-related functions
)

// standardModules returns a copy of the Starlark universe with additional modules
// This is used by both the compiler and evaluator to ensure consistency between
// compilation-time checks and runtime execution
func standardModules() starlarkLib.StringDict {
	// Clone the universe to avoid modifying the global one
	universe := maps.Clone(starlarkLib.Universe)

	// Add additional modules
	universe[namespaceJSON] = starlarkJSON.Module
	universe[namespaceMath] = starlarkMath.Module
	universe[namespaceTime] = starlarkTime.Module

	return universe
}
