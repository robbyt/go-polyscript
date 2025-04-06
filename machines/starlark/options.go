package starlark

import (
	"fmt"

	"github.com/robbyt/go-polyscript/engine/options"
	"github.com/robbyt/go-polyscript/machines/types"
)

// StarlarkOptions provides configuration options for the Starlark compiler
type StarlarkOptions struct {
	Globals []string
}

// GetGlobals returns the list of global variables available to scripts
func (o *StarlarkOptions) GetGlobals() []string {
	return o.Globals
}

// WithGlobals creates an option to set the globals for Starlark scripts
func WithGlobals(globals []string) options.Option {
	return func(c *options.Config) error {
		// Validate this is being used with the correct machine type
		if c.GetMachineType() != types.Starlark && c.GetMachineType() != "" {
			return fmt.Errorf(
				"WithGlobals for Starlark can only be used with Starlark machine, got %s",
				c.GetMachineType(),
			)
		}

		// Set the machine-specific options
		c.SetCompilerOptions(&StarlarkOptions{
			Globals: globals,
		})
		return nil
	}
}
