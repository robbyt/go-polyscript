package options

import (
	"fmt"
	"log/slog"

	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/machines/types"
)

// Config holds all configuration for creating a script engine
type Config struct {
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

// Option is a function that modifies Config
type Option func(*Config) error

// WithLogger sets the logger for the script engine
func WithLogger(handler slog.Handler) Option {
	return func(c *Config) error {
		if handler != nil {
			c.handler = handler
		}
		return nil
	}
}

// WithDataProvider sets the data provider for the script engine
func WithDataProvider(provider data.InputDataProvider) Option {
	return func(c *Config) error {
		if provider != nil {
			c.dataProvider = provider
		}
		return nil
	}
}

// WithLoader sets the script loader
func WithLoader(l loader.Loader) Option {
	return func(c *Config) error {
		if l != nil {
			c.loader = l
		}
		return nil
	}
}

// Validate performs basic validation on the configuration
func (c *Config) Validate() error {
	if c.loader == nil {
		return fmt.Errorf("no loader specified")
	}
	if c.machineType == "" {
		return fmt.Errorf("no machine type specified")
	}
	return nil
}

// GetHandler returns the configured logger
func (c *Config) GetHandler() slog.Handler {
	return c.handler
}

// SetHandler sets the logger
func (c *Config) SetHandler(handler slog.Handler) {
	c.handler = handler
}

// GetMachineType returns the configured machine type
func (c *Config) GetMachineType() types.Type {
	return c.machineType
}

// SetMachineType sets the machine type
func (c *Config) SetMachineType(machineType types.Type) {
	c.machineType = machineType
}

// GetDataProvider returns the configured data provider
func (c *Config) GetDataProvider() data.InputDataProvider {
	return c.dataProvider
}

// SetDataProvider sets the data provider
func (c *Config) SetDataProvider(provider data.InputDataProvider) {
	c.dataProvider = provider
}

// GetLoader returns the configured loader
func (c *Config) GetLoader() loader.Loader {
	return c.loader
}

// GetCompilerOptions returns the machine-specific compiler options
func (c *Config) GetCompilerOptions() any {
	return c.compilerOptions
}

// SetCompilerOptions sets the machine-specific compiler options
func (c *Config) SetCompilerOptions(options any) {
	c.compilerOptions = options
}
