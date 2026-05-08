package compiler

import (
	"log/slog"
	"slices"

	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/platform/constants"
)

// FunctionalOption is a function that configures a Compiler instance
type FunctionalOption func(*Compiler) error

// WithGlobals creates an option to set the globals for Starlark scripts
func WithGlobals(globals []string) FunctionalOption {
	return func(c *Compiler) error {
		c.globals = globals
		return nil
	}
}

// WithCtxGlobal is a convenience option to set the user-specified global to 'ctx'
func WithCtxGlobal() FunctionalOption {
	return func(c *Compiler) error {
		if len(c.globals) == 0 {
			c.globals = []string{constants.Ctx}
		} else if !slices.Contains(c.globals, constants.Ctx) {
			c.globals = append(c.globals, constants.Ctx)
		}
		return nil
	}
}

// WithLogHandler creates an option to set the log handler for Starlark compiler.
// This is the preferred option for logging configuration as it provides
// more flexibility through the slog.Handler interface.
//
// Passing a nil handler is a no-op equivalent to not calling the option:
// the compiler inherits from slog.Default() via [helpers.SetupLogger].
func WithLogHandler(handler slog.Handler) FunctionalOption {
	return func(c *Compiler) error {
		if handler == nil {
			return nil
		}
		c.logHandler = handler
		// Clear logger if handler is explicitly set
		c.logger = nil
		return nil
	}
}

// WithLogger creates an option to set a specific logger for Starlark compiler.
// This is less flexible than WithLogHandler but allows users to customize
// their logging group configuration.
//
// Passing a nil logger is a no-op equivalent to not calling the option.
func WithLogger(logger *slog.Logger) FunctionalOption {
	return func(c *Compiler) error {
		if logger == nil {
			return nil
		}
		c.logger = logger
		// Clear handler if logger is explicitly set
		c.logHandler = nil
		return nil
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
		c.logHandler, c.logger = helpers.SetupLogger(c.logHandler, "starlark", "Compiler")
	}
}

// validate checks if the compiler configuration is valid.
func (c *Compiler) validate() error {
	return nil
}

// applyDefaults sets the default values for a compiler.
func (c *Compiler) applyDefaults() {
	if c.globals == nil {
		c.globals = []string{}
	}
}
