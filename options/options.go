package options

import (
	"fmt"
	"log/slog"

	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/machines/types"
)

// Config holds all configuration for creating a script engine
// It is not exported as it should only be modified via Option functions
type config struct {
	// Logger for the engine
	handler slog.Handler
	// Type of machine to use (starlark, risor, extism)
	machineType types.Type
	// Data provider for passing values to the script
	dataProvider data.InputDataProvider
	// Loader for the script content
	loader loader.Loader
	// Machine-specific options
	compilerOptions any
}

// Option is a function that modifies config
type Option func(*config) error

// WithLogger sets the logger for the script engine
func WithLogger(handler slog.Handler) Option {
	return func(c *config) error {
		if handler != nil {
			c.handler = handler
		}
		return nil
	}
}

// WithDataProvider sets the data provider for the script engine
func WithDataProvider(provider data.InputDataProvider) Option {
	return func(c *config) error {
		if provider != nil {
			c.dataProvider = provider
		}
		return nil
	}
}

// WithLoader sets the script loader
func WithLoader(l loader.Loader) Option {
	return func(c *config) error {
		if l != nil {
			c.loader = l
		}
		return nil
	}
}

// validate performs basic validation on the configuration
func (c *config) validate() error {
	if c.loader == nil {
		return fmt.Errorf("no loader specified")
	}
	if c.machineType == "" {
		return fmt.Errorf("no machine type specified")
	}
	return nil
}

// GetHandler returns the configured logger
func (c *config) GetHandler() slog.Handler {
	return c.handler
}

// GetMachineType returns the configured machine type
func (c *config) GetMachineType() types.Type {
	return c.machineType
}

// GetDataProvider returns the configured data provider
func (c *config) GetDataProvider() data.InputDataProvider {
	return c.dataProvider
}

// GetLoader returns the configured loader
func (c *config) GetLoader() loader.Loader {
	return c.loader
}

// GetCompilerOptions returns the machine-specific compiler options
func (c *config) GetCompilerOptions() any {
	return c.compilerOptions
}
