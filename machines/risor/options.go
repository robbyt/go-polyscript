package risor

import (
	"fmt"
	"log/slog"
	"os"
	"slices"

	"github.com/robbyt/go-polyscript/execution/constants"
)

// compilerConfig holds the configuration for the Risor compiler
type compilerConfig struct {
	Globals    []string
	LogHandler slog.Handler
	Logger     *slog.Logger
}

// Option is a function that configures a compilerConfig
type Option func(*compilerConfig) error

// WithGlobals creates an option to set the globals for Risor scripts
func WithGlobals(globals []string) Option {
	return func(cfg *compilerConfig) error {
		cfg.Globals = globals
		return nil
	}
}

// WithCtxGlobal is a convenience option to set the user-specified global to 'ctx'
func WithCtxGlobal() Option {
	return func(cfg *compilerConfig) error {
		if !slices.Contains(cfg.Globals, constants.Ctx) {
			cfg.Globals = append(cfg.Globals, constants.Ctx)
		}
		return nil
	}
}

// WithLogHandler creates an option to set the log handler for Risor compiler.
// This is the preferred option for logging configuration as it provides
// more flexibility through the slog.Handler interface.
func WithLogHandler(handler slog.Handler) Option {
	return func(cfg *compilerConfig) error {
		if handler == nil {
			return fmt.Errorf("log handler cannot be nil")
		}
		cfg.LogHandler = handler
		// Clear logger if handler is explicitly set
		cfg.Logger = nil
		return nil
	}
}

// WithLogger creates an option to set a specific logger for Risor compiler.
// This is less flexible than WithLogHandler but allows users to customize
// their logging group configuration.
func WithLogger(logger *slog.Logger) Option {
	return func(cfg *compilerConfig) error {
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
func applyDefaults(cfg *compilerConfig) {
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
func validate(cfg *compilerConfig) error {
	// Ensure we have either a logger or a handler
	if cfg.LogHandler == nil && cfg.Logger == nil {
		return fmt.Errorf("either log handler or logger must be specified")
	}

	return nil
}
