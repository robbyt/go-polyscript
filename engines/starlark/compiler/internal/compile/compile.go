package compile

import (
	"fmt"

	"github.com/robbyt/go-polyscript/engines/starlark/internal"
	starlarkLib "go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// compile parses and compiles the script content into a Starlark program
func compile(
	scriptBodyBytes []byte,
	opts *syntax.FileOptions,
	globals starlarkLib.StringDict,
) (*starlarkLib.Program, error) {
	if scriptBodyBytes == nil {
		return nil, ErrContentNil
	}

	if opts == nil {
		opts = &syntax.FileOptions{}
	}

	// Create merged dictionary with standard modules and provided globals
	mergedGlobals := make(starlarkLib.StringDict)

	// Start with standard modules (universe plus additional modules)
	for k, v := range internal.StarlarkModules() {
		mergedGlobals[k] = v
	}

	// Then add provided globals, allowing them to override defaults
	for k, v := range globals {
		mergedGlobals[k] = v
	}

	// Parse and compile the script
	f, err := opts.Parse("", scriptBodyBytes, 0)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCompileFailed, err)
	}

	prog, err := starlarkLib.FileProgram(f, mergedGlobals.Has)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCompileFailed, err)
	}

	return prog, nil
}

// CompileWithEmptyGlobals parses and compiles the script content, with custom global names
// which are needed when parsing a script that will eventually have globals injected at eval time.
// For example, if a script uses a request or response object, it needs to be compiled with those
// global names, even though they won't be available until eval time.
func CompileWithEmptyGlobals(
	scriptBodyBytes []byte,
	globals []string,
) (*starlarkLib.Program, error) {
	// Set up FileOptions with globals reassignment enabled
	opts := &syntax.FileOptions{
		GlobalReassign: true, // Allow later reassignment of the globals we're injecting right now
	}

	// Get our standard modules to avoid "undefined" errors
	stdModules := internal.StarlarkModules()

	// Create a StringDict with the provided globals as None values
	predeclared := make(starlarkLib.StringDict, len(globals))
	for _, name := range globals {
		// Skip if this is already in our standard modules
		if stdModules.Has(name) {
			continue
		}
		predeclared[name] = starlarkLib.None
	}

	return compile(scriptBodyBytes, opts, predeclared)
}
