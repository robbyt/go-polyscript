package starlark

import (
	"fmt"
	"log/slog"
	"os"
	"slices"

	"github.com/robbyt/go-polyscript/execution/constants"
)

// compilerOptions holds the configuration for the Starlark compiler
type compilerOptions struct {
	Globals    []string
	LogHandler slog.Handler
	Logger     *slog.Logger
}

// CompilerOption is a function that configures a compilerOptions instance
type CompilerOption func(*compilerOptions) error

// WithGlobals creates an option to set the globals for Starlark scripts
func WithGlobals(globals []string) CompilerOption {
	return func(cfg *compilerOptions) error {
		cfg.Globals = globals
		return nil
	}
}

// WithCtxGlobal is a convenience option to set the user-specified global to 'ctx'
func WithCtxGlobal() CompilerOption {
	return func(cfg *compilerOptions) error {
		if !slices.Contains(cfg.Globals, constants.Ctx) {
			cfg.Globals = append(cfg.Globals, constants.Ctx)
		}
		return nil
	}
}

// WithLogHandler creates an option to set the log handler for Starlark compiler.
// This is the preferred option for logging configuration as it provides
// more flexibility through the slog.Handler interface.
func WithLogHandler(handler slog.Handler) CompilerOption {
	return func(cfg *compilerOptions) error {
		if handler == nil {
			return fmt.Errorf("log handler cannot be nil")
		}
		cfg.LogHandler = handler
		// Clear logger if handler is explicitly set
		cfg.Logger = nil
		return nil
	}
}

// WithLogger creates an option to set a specific logger for Starlark compiler.
// This is less flexible than WithLogHandler but allows users to customize
// their logging group configuration.
func WithLogger(logger *slog.Logger) CompilerOption {
	return func(cfg *compilerOptions) error {
		if logger == nil {
			return fmt.Errorf("logger cannot be nil")
		}
		cfg.Logger = logger
		// Clear handler if logger is explicitly set
		cfg.LogHandler = nil
		return nil
	}
}

// applyDefaults sets the default values for a compilerConfig
func applyDefaults(cfg *compilerOptions) {
	// Default to stderr for logging if neither handler nor logger specified
	if cfg.LogHandler == nil && cfg.Logger == nil {
		cfg.LogHandler = slog.NewTextHandler(os.Stderr, nil)
	}

	// Default to empty globals if not specified
	if cfg.Globals == nil {
		cfg.Globals = []string{}
	}
}

// validate checks if the configuration is valid
func validate(cfg *compilerOptions) error {
	// Ensure we have either a logger or a handler
	if cfg.LogHandler == nil && cfg.Logger == nil {
		return fmt.Errorf("either log handler or logger must be specified")
	}

	return nil
}
