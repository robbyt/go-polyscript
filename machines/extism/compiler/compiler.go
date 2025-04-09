package compiler

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/machines/extism/compiler/internal/compile"
)

// Compiler implements the script.Compiler interface for WASM modules
type Compiler struct {
	entryPointName string
	ctx            context.Context
	options        *compile.Settings
	logHandler     slog.Handler
	logger         *slog.Logger
}

// NewCompiler creates a new Extism WASM Compiler instance with the provided options.
func NewCompiler(opts ...FunctionalOption) (*Compiler, error) {
	// Initialize compiler with empty values
	c := &Compiler{
		entryPointName: defaultEntryPoint,
		options:        &compile.Settings{},
	}

	// Apply defaults
	c.applyDefaults()

	// Apply all options
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, fmt.Errorf("error applying compiler option: %w", err)
		}
	}

	// Validate the configuration
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("invalid compiler configuration: %w", err)
	}

	// Set up logging based on provided options
	if c.logger != nil {
		// User provided a custom logger
		c.logHandler = c.logger.Handler()
	} else {
		// User provided a handler or we're using the default
		c.logHandler, c.logger = helpers.SetupLogger(c.logHandler, "extism", "Compiler")
	}

	return c, nil
}

func (c *Compiler) String() string {
	return "extism.Compiler"
}

// Compile implements script.Compiler
// TODO: Some error paths are difficult to test with the current design
// Consider adding integration tests for hard-to-reach error cases.
func (c *Compiler) Compile(scriptReader io.ReadCloser) (script.ExecutableContent, error) {
	logger := c.logger.WithGroup("compile")

	if scriptReader == nil {
		return nil, ErrContentNil
	}

	scriptBytes, err := io.ReadAll(scriptReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read script: %w", err)
	}

	err = scriptReader.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close reader: %w", err)
	}

	if len(scriptBytes) == 0 {
		logger.Error("Compile called with empty script")
		return nil, ErrContentNil
	}

	logger.Debug("Starting WASM compilation", "scriptLength", len(scriptBytes))

	// Compile the WASM module using the CompileBytes function
	plugin, err := compile.CompileBytes(c.ctx, scriptBytes, c.options)
	if err != nil {
		logger.Warn("WASM compilation failed", "error", err)
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	if plugin == nil {
		logger.Error("Compilation returned nil plugin")
		return nil, ErrBytecodeNil
	}

	// Create a temporary instance to verify the entry point exists
	instance, err := plugin.Instance(c.ctx, extismSDK.PluginInstanceConfig{})
	if err != nil {
		logger.Error("Failed to create test instance", "error", err)
		return nil, fmt.Errorf("%w: failed to create test instance: %w", ErrValidationFailed, err)
	}
	defer func() {
		if err := instance.Close(c.ctx); err != nil {
			c.logger.Warn("Failed to close Extism plugin instance in compiler", "error", err)
		}
	}()

	// Verify the entry point function exists
	funcName := c.GetEntryPointName()
	if !instance.FunctionExists(funcName) {
		logger.Error("Entry point function not found", "function", funcName)
		return nil, fmt.Errorf(
			"%w: entry point function '%s' not found",
			ErrValidationFailed,
			funcName,
		)
	}

	// Create executable with the compiled plugin
	executable := NewExecutable(scriptBytes, plugin, funcName)
	if executable == nil {
		logger.Warn("Failed to create Executable from WASM plugin")
		return nil, ErrExecCreationFailed
	}

	logger.Debug("WASM compilation completed successfully")
	return executable, nil
}

// GetEntryPointName is a getter for the func name entrypoint
func (c *Compiler) GetEntryPointName() string {
	return c.entryPointName
}
