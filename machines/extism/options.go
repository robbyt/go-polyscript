package extism

import (
	"fmt"

	"github.com/robbyt/go-polyscript/machines/types"
	"github.com/robbyt/go-polyscript/options"
)

// ExtismOptions provides configuration options for the Extism compiler
type ExtismOptions struct {
	EntryPoint string
}

// GetEntryPointName returns the WASM entry point function name
func (o *ExtismOptions) GetEntryPointName() string {
	return o.EntryPoint
}

// WithEntryPoint creates an option to set the entry point for Extism WASM modules
func WithEntryPoint(entryPoint string) options.Option {
	return func(c *options.Config) error {
		// Validate this is being used with the correct machine type
		if c.GetMachineType() != types.Extism && c.GetMachineType() != "" {
			return fmt.Errorf(
				"WithEntryPoint can only be used with Extism machine, got %s",
				c.GetMachineType(),
			)
		}

		// Set the machine-specific options
		c.SetCompilerOptions(&ExtismOptions{
			EntryPoint: entryPoint,
		})
		return nil
	}
}
