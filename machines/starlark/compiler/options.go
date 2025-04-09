package compiler

import (
	"fmt"
	"log/slog"
	"os"
	"slices"

	"github.com/robbyt/go-polyscript/execution/constants"
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

// WithLogger creates an option to set a specific logger for Starlark compiler.
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

// validate checks if the compiler configuration is valid
func (c *Compiler) validate() error {
	// Ensure we have either a logger or a handler
	if c.logHandler == nil && c.logger == nil {
		return fmt.Errorf("either log handler or logger must be specified")
	}

	return nil
}

// applyDefaults sets the default values for a compiler
func (c *Compiler) applyDefaults() {
	// Default to stderr for logging if neither handler nor logger specified
	if c.logHandler == nil && c.logger == nil {
		c.logHandler = slog.NewTextHandler(os.Stderr, nil)
	}

	// Default to empty globals if not specified
	if c.globals == nil {
		c.globals = []string{}
	}
}
