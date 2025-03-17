package starlark

// BasicCompilerOptions provides configuration options for the Starlark compiler
type BasicCompilerOptions struct {
	Globals []string
}

// GetGlobals returns the list of global variables available to scripts
func (o *BasicCompilerOptions) GetGlobals() []string {
	return o.Globals
}
