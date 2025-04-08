package extism

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/internal/helpers"
)

const defaultEntryPoint = "main"

// Compiler implements the script.Compiler interface for WASM modules
type Compiler struct {
	entryPointName atomic.Value
	ctx            context.Context
	options        *compileOptions
	logHandler     slog.Handler
	logger         *slog.Logger
}

type CompilerOptions interface {
	GetEntryPointName() string
}

// NewCompiler creates a new Extism WASM Compiler instance. External config of the compiler not
// currently supported.
func NewCompiler(handler slog.Handler, compilerOptions CompilerOptions) *Compiler {
	handler, logger := helpers.SetupLogger(handler, "extism", "Compiler")

	entryPointName := compilerOptions.GetEntryPointName()
	if entryPointName == "" {
		entryPointName = defaultEntryPoint
	}

	var funcName atomic.Value
	funcName.Store(entryPointName)

	return &Compiler{
		entryPointName: funcName,
		ctx:            context.Background(),
		options:        withDefaultCompileOptions(),
		logHandler:     handler,
		logger:         logger,
	}
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
	plugin, err := CompileBytes(c.ctx, scriptBytes, c.options)
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
	executable := newExecutable(scriptBytes, plugin, funcName)
	if executable == nil {
		logger.Warn("Failed to create Executable from WASM plugin")
		return nil, ErrExecCreationFailed
	}

	logger.Debug("WASM compilation completed successfully")
	return executable, nil
}

// SetEntryPointName is a way to point the compiler at a different entrypoint in the wasm binary
func (c *Compiler) SetEntryPointName(fName string) {
	c.entryPointName.Store(fName)
}

// GetEntryPointName is a getter for the func name entrypoint
func (c *Compiler) GetEntryPointName() string {
	return c.entryPointName.Load().(string)
}
