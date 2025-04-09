package compiler

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/machines/extism/compiler/internal/compile"
	"github.com/tetratelabs/wazero"
)

// FunctionalOption is a function that configures a Compiler instance
type FunctionalOption func(*Compiler) error

// WithEntryPoint creates an option to set the entry point for Extism WASM modules
func WithEntryPoint(entryPoint string) FunctionalOption {
	return func(c *Compiler) error {
		if entryPoint == "" {
			return fmt.Errorf("entry point cannot be empty")
		}
		c.entryPointName = entryPoint
		return nil
	}
}

// WithLogHandler creates an option to set the log handler for Extism compiler.
// This is the preferred option for logging configuration as it provides
// more flexibility through the slog.Handler interface.
func WithLogHandler(handler slog.Handler) FunctionalOption {
	return func(c *Compiler) error {
		if handler == nil {
			return fmt.Errorf("log handler cannot be nil")
		}
		c.logHandler = handler
		// Clear logger if handler is explicitly set
		c.logger = nil
		return nil
	}
}

// WithLogger creates an option to set a specific logger for Extism compiler.
// This is less flexible than WithLogHandler but allows users to customize
// their logging group configuration.
func WithLogger(logger *slog.Logger) FunctionalOption {
	return func(c *Compiler) error {
		if logger == nil {
			return fmt.Errorf("logger cannot be nil")
		}
		c.logger = logger
		// Clear handler if logger is explicitly set
		c.logHandler = nil
		return nil
	}
}

// WithWASIEnabled creates an option to enable or disable WASI support
func WithWASIEnabled(enabled bool) FunctionalOption {
	return func(c *Compiler) error {
		if c.options == nil {
			c.options = &compile.Settings{}
		}
		c.options.EnableWASI = enabled
		return nil
	}
}

// WithRuntimeConfig creates an option to set a custom wazero runtime configuration
func WithRuntimeConfig(config wazero.RuntimeConfig) FunctionalOption {
	return func(c *Compiler) error {
		if config == nil {
			return fmt.Errorf("runtime config cannot be nil")
		}
		if c.options == nil {
			c.options = &compile.Settings{}
		}
		c.options.RuntimeConfig = config
		return nil
	}
}

// WithHostFunctions creates an option to set additional host functions
func WithHostFunctions(funcs []extismSDK.HostFunction) FunctionalOption {
	return func(c *Compiler) error {
		if c.options == nil {
			c.options = &compile.Settings{}
		}
		c.options.HostFunctions = funcs
		return nil
	}
}

// WithContext creates an option to set a custom context for the Extism compiler.
func WithContext(ctx context.Context) FunctionalOption {
	return func(c *Compiler) error {
		if ctx == nil {
			return fmt.Errorf("context cannot be nil")
		}
		c.ctx = ctx
		return nil
	}
}

// applyDefaults sets the default values for a compiler
func (c *Compiler) applyDefaults() {
	// Default to stderr for logging if neither handler nor logger specified
	if c.logHandler == nil && c.logger == nil {
		c.logHandler = slog.NewTextHandler(os.Stderr, nil)
	}

	// Set default entry point
	if c.entryPointName == "" {
		c.entryPointName = defaultEntryPoint
	}

	// Initialize options struct if nil
	if c.options == nil {
		c.options = &compile.Settings{}
	}

	// Set default runtime config if not already set
	if c.options.RuntimeConfig == nil {
		c.options.RuntimeConfig = wazero.NewRuntimeConfig()
	}

	// Set default host functions if not already set
	if c.options.HostFunctions == nil {
		c.options.HostFunctions = []extismSDK.HostFunction{}
	}

	// Default WASI to true (EnableWASI is a bool so we don't need to check if it's nil)
	c.options.EnableWASI = true

	// Default context
	if c.ctx == nil {
		c.ctx = context.Background()
	}
}

// setupLogger configures the logger and handler based on the current state.
// This is idempotent and can be called multiple times during initialization.
func (c *Compiler) setupLogger() {
	if c.logger != nil {
		// When a logger is explicitly set, extract its handler
		c.logHandler = c.logger.Handler()
	} else {
		// Otherwise use the handler (which might be default or custom) to create the logger
		c.logHandler, c.logger = helpers.SetupLogger(c.logHandler, "extism", "Compiler")
	}
}

// validate checks if the compiler configuration is valid
func (c *Compiler) validate() error {
	// Ensure we have either a logger or a handler
	if c.logHandler == nil && c.logger == nil {
		return fmt.Errorf("either log handler or logger must be specified")
	}

	// Entry point must be non-empty
	if c.entryPointName == "" {
		return fmt.Errorf("entry point must be specified")
	}

	// Runtime config cannot be nil
	if c.options == nil || c.options.RuntimeConfig == nil {
		return fmt.Errorf("runtime config cannot be nil")
	}

	// Context cannot be nil
	if c.ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	return nil
}
