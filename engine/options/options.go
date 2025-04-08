package options

import (
	"errors"
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
	dataProvider data.Provider
	// Loader for the script content
	loader loader.Loader
}

// Option is a function that modifies Config
type Option func(*Config) error

// WithLogHandler sets the logger for the script engine
func WithLogHandler(handler slog.Handler) Option {
	return func(c *Config) error {
		if handler != nil {
			c.handler = handler
		}
		return nil
	}
}

// WithSlog sets the slog logger for the script engine
func WithSlog(logger *slog.Logger) Option {
	return func(c *Config) error {
		if logger != nil {
			c.handler = logger.Handler()
		}
		return nil
	}
}

// WithDataProvider sets the data provider for the script engine
func WithDataProvider(provider data.Provider) Option {
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

// Validate performs basic validation on the common configuration. Machine-specific
// validation is performed in each machine-specific VM package.
func (c *Config) Validate() error {
	var errz []error
	if c.handler == nil {
		errz = append(errz, fmt.Errorf("no logger specified"))
	}
	if c.machineType == "" {
		errz = append(errz, fmt.Errorf("no machine type specified"))
	}
	if c.dataProvider == nil {
		errz = append(errz, fmt.Errorf("no data provider specified"))
	}
	if c.loader == nil {
		errz = append(errz, fmt.Errorf("no loader specified"))
	}
	return errors.Join(errz...)
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
func (c *Config) GetDataProvider() data.Provider {
	return c.dataProvider
}

// SetDataProvider sets the data provider
func (c *Config) SetDataProvider(provider data.Provider) {
	c.dataProvider = provider
}

// GetLoader returns the configured loader
func (c *Config) GetLoader() loader.Loader {
	return c.loader
}

// SetLoader sets the loader
func (c *Config) SetLoader(l loader.Loader) {
	c.loader = l
}

// Type Conversion Helpers (used internally, not part of the public API)
// These functions help polyscript.go handle the correct conversion between engine options and machine options

// This file no longer contains machine-specific option wrappers.
// Machine-specific options should be used directly from their respective packages:
// - extismMachine.WithEntryPoint()
// - risorMachine.WithGlobals()
// - risorMachine.WithCtxGlobal()
// - starlarkMachine.WithGlobals()
// - starlarkMachine.WithCtxGlobal()
