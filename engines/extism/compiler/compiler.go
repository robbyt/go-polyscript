package compiler

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/engines/extism/adapters"
	"github.com/robbyt/go-polyscript/engines/extism/compiler/internal/compile"
	"github.com/robbyt/go-polyscript/platform/script"
)

// compileFunc is the signature for compiling raw WASM bytes into a CompiledPlugin.
// Defaulted to compile.CompileBytes; overridable in tests to drive failure paths.
type compileFunc func(ctx context.Context, wasmBytes []byte, opts *compile.Settings) (adapters.CompiledPlugin, error)

// Compiler implements the script.Compiler interface for WASM modules
type Compiler struct {
	entryPointName string
	ctx            context.Context
	options        *compile.Settings
	logHandler     slog.Handler
	logger         *slog.Logger
	compileFn      compileFunc
}

// New creates a new Extism WASM Compiler instance with the provided options.
func New(opts ...FunctionalOption) (*Compiler, error) {
	// Initialize the compiler with an empty struct
	c := &Compiler{}

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

	// Finalize logger setup after all options have been applied
	c.setupLogger()

	return c, nil
}

func (c *Compiler) String() string {
	return "extism.Compiler"
}

// Compile implements script.Compiler
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

	// Compile the WASM module using the configured compile function (defaults to compile.CompileBytes)
	plugin, err := c.compileFn(c.ctx, scriptBytes, c.options)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	if plugin == nil {
		return nil, ErrBytecodeNil
	}

	// Create a temporary instance to verify the entry point exists
	instance, err := plugin.Instance(c.ctx, extismSDK.PluginInstanceConfig{})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create test instance: %w", ErrValidationFailed, err)
	}
	defer func() {
		if err := instance.Close(c.ctx); err != nil {
			logger.Warn("Failed to close Extism plugin instance in compiler", "error", err)
		}
	}()

	// Verify the entry point function exists
	funcName := c.GetEntryPointName()
	if !instance.FunctionExists(funcName) {
		return nil, fmt.Errorf(
			"%w: entry point function '%s' not found",
			ErrValidationFailed,
			funcName,
		)
	}

	// Create executable with the compiled plugin. Inputs are guaranteed non-empty
	// by the checks above (scriptBytes, plugin) and validate() (funcName), so
	// NewExecutable cannot return nil here.
	executable := NewExecutable(scriptBytes, plugin, funcName)

	logger.Debug("WASM compilation completed")
	return executable, nil
}

// GetEntryPointName is a getter for the func name entrypoint
func (c *Compiler) GetEntryPointName() string {
	return c.entryPointName
}
