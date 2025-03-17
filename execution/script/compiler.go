package script

import "io"

// Compiler defines the interface for validating scripts before execution.
// It checks syntax and semantics, and may perform parsing, compilation,
// and optimization. A valid script is returned as ExecutableContent.
//
// Example usage:
//
//	var comp Compiler = NewRisorCompiler(globals)
//	executableContent, err := comp.Compile(&scriptContent)
//	if err != nil {
//	    // Handle validation error
//	}
//	// Use executableContent for execution
type Compiler interface {
	// Compile checks if a script is valid and returns it as executable content.
	// The returned ExecutableContent contains the validated and possibly compiled
	// script ready for execution.
	//
	// Parameters:
	//   - script: A pointer to the script content string
	//
	// Returns:
	//   - ExecutableContent: The validated script
	//   - error: Details about validation failures (syntax errors, undefined globals)
	Compile(scriptReader io.ReadCloser) (ExecutableContent, error)
}
