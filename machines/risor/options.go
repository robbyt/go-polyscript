package risor

// CompilerOptions provides configuration options for the Risor compiler
type BasicCompilerOptions struct {
	Globals []string
}

// GetGlobals returns the list of global variables available to scripts
func (o *BasicCompilerOptions) GetGlobals() []string {
	return o.Globals
}
