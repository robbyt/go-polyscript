package extism

import (
	"fmt"
	"log/slog"
	"os"

	extismSDK "github.com/extism/go-sdk"
	"github.com/tetratelabs/wazero"
)

// compilerConfig holds the configuration for the Extism compiler
type compilerConfig struct {
	EntryPoint    string
	LogHandler    slog.Handler
	Logger        *slog.Logger
	EnableWASI    bool
	RuntimeConfig wazero.RuntimeConfig
	HostFunctions []extismSDK.HostFunction
}

// Option is a function that configures a compilerConfig
type Option func(*compilerConfig) error

// WithEntryPoint creates an option to set the entry point for Extism WASM modules
func WithEntryPoint(entryPoint string) Option {
	return func(cfg *compilerConfig) error {
		if entryPoint == "" {
			return fmt.Errorf("entry point cannot be empty")
		}
		cfg.EntryPoint = entryPoint
		return nil
	}
}

// WithLogHandler creates an option to set the log handler for Extism compiler.
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

// WithLogger creates an option to set a specific logger for Extism compiler.
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

// WithWASIEnabled creates an option to enable or disable WASI support
func WithWASIEnabled(enabled bool) Option {
	return func(cfg *compilerConfig) error {
		cfg.EnableWASI = enabled
		return nil
	}
}

// WithRuntimeConfig creates an option to set a custom wazero runtime configuration
func WithRuntimeConfig(config wazero.RuntimeConfig) Option {
	return func(cfg *compilerConfig) error {
		if config == nil {
			return fmt.Errorf("runtime config cannot be nil")
		}
		cfg.RuntimeConfig = config
		return nil
	}
}

// WithHostFunctions creates an option to set additional host functions
func WithHostFunctions(funcs []extismSDK.HostFunction) Option {
	return func(cfg *compilerConfig) error {
		cfg.HostFunctions = funcs
		return nil
	}
}

// applyDefaults sets the default values for a compilerConfig
func applyDefaults(cfg *compilerConfig) {
	// Default to stderr for logging if neither handler nor logger specified
	if cfg.LogHandler == nil && cfg.Logger == nil {
		cfg.LogHandler = slog.NewTextHandler(os.Stderr, nil)
	}

	// Default entry point
	if cfg.EntryPoint == "" {
		cfg.EntryPoint = defaultEntryPoint
	}

	// Default WASI setting
	cfg.EnableWASI = true

	// Default runtime config
	if cfg.RuntimeConfig == nil {
		cfg.RuntimeConfig = wazero.NewRuntimeConfig()
	}

	// Default to empty host functions
	if cfg.HostFunctions == nil {
		cfg.HostFunctions = []extismSDK.HostFunction{}
	}
}

// validate checks if the configuration is valid
func validate(cfg *compilerConfig) error {
	// Ensure we have either a logger or a handler
	if cfg.LogHandler == nil && cfg.Logger == nil {
		return fmt.Errorf("either log handler or logger must be specified")
	}

	// Entry point must be non-empty
	if cfg.EntryPoint == "" {
		return fmt.Errorf("entry point must be specified")
	}

	// Runtime config cannot be nil
	if cfg.RuntimeConfig == nil {
		return fmt.Errorf("runtime config cannot be nil")
	}

	return nil
}
