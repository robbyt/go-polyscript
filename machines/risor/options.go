package risor

import (
	"fmt"

	"github.com/robbyt/go-polyscript/engine/options"
	"github.com/robbyt/go-polyscript/machines/types"
)

// RisorOptions provides configuration options for the Risor compiler
type RisorOptions struct {
	Globals []string
}

// GetGlobals returns the list of global variables available to scripts
func (o *RisorOptions) GetGlobals() []string {
	return o.Globals
}

// WithGlobals creates an option to set the globals for Risor scripts
func WithGlobals(globals []string) options.Option {
	return func(c *options.Config) error {
		// Validate this is being used with the correct machine type
		if c.GetMachineType() != types.Risor && c.GetMachineType() != "" {
			return fmt.Errorf(
				"WithGlobals for Risor can only be used with Risor machine, got %s",
				c.GetMachineType(),
			)
		}

		// Set the machine-specific options
		c.SetCompilerOptions(&RisorOptions{
			Globals: globals,
		})
		return nil
	}
}
