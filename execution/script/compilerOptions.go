package script

// CompilerOptions is a struct containing various values used by the various compiler types. Some
// values may only be useful for some compilers.
type CompilerOptions struct {
	// Globals is a list of top-level global variables that are available to the script during parse time.
	Globals []string

	// EntryPoint is the entry point for the script. This is the function that will be called
	// when the script is executed and is usually the main function.
	EntryPointName string
}

// GetGlobals returns the list of global variables for the compiler.
func (co CompilerOptions) GetGlobals() []string {
	return co.Globals
}

// GetEntryPoint returns the entry point for the compiler.
func (co CompilerOptions) GetEntryPointName() string {
	return co.EntryPointName
}
