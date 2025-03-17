package extism

// BasicCompilerOptions provides configuration options for the Extism compiler
type BasicCompilerOptions struct {
	EntryPoint string
}

// GetEntryPointName returns the WASM entry point function name
func (o *BasicCompilerOptions) GetEntryPointName() string {
	return o.EntryPoint
}
