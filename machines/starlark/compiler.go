package starlark

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript/execution/script"
)

type Compiler struct {
	globals    []string
	logHandler slog.Handler
	logger     *slog.Logger
}

type CompilerOptions interface {
	GetGlobals() []string
}

// NewCompiler creates a new Starlark-specific Compiler instance with the provided global variables.
// Global variables are used during script parsing to validate global name usage.
func NewCompiler(handler slog.Handler, compilerOptions CompilerOptions) *Compiler {
	if handler == nil {
		defaultHandler := slog.NewTextHandler(os.Stdout, nil)
		handler = defaultHandler.WithGroup("starlark")
		// Create a logger from the handler rather than using slog directly
		defaultLogger := slog.New(handler)
		defaultLogger.Warn("Handler is nil, using the default logger configuration.")
	}

	return &Compiler{
		globals:    compilerOptions.GetGlobals(),
		logHandler: handler,
		logger:     slog.New(handler.WithGroup("Compiler")),
	}
}

func (c *Compiler) getLogger() *slog.Logger {
	return c.logger
}

func (c *Compiler) compile(scriptBodyBytes []byte) (*Executable, error) {
	logger := c.getLogger()
	if len(scriptBodyBytes) == 0 {
		logger.Error("Compile called with nil script")
		return nil, ErrContentNil
	}

	logger.Debug("Starting validation")

	// Compile the script with globals
	program, err := compileWithEmptyGlobals(scriptBodyBytes, c.globals)
	if err != nil {
		logger.Warn("Compilation failed", "error", err)
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	if program == nil {
		logger.Error("Compilation returned nil program")
		return nil, ErrBytecodeNil
	}

	// Create executable with the compiled program
	starlarkExec := NewExecutable(scriptBodyBytes, program)
	if starlarkExec == nil {
		logger.Warn("Failed to create Executable from program")
		return nil, ErrExecCreationFailed
	}

	logger.Debug("Validation completed")
	return starlarkExec, nil
}

func (c *Compiler) String() string {
	return "starlark.Compiler"
}

// Compile turns the provided script content into runnable bytecode.
func (c *Compiler) Compile(scriptReader io.ReadCloser) (script.ExecutableContent, error) {
	if scriptReader == nil {
		return nil, ErrContentNil
	}

	scriptBodyBytes, err := io.ReadAll(scriptReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read script: %w", err)
	}

	err = scriptReader.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close reader: %w", err)
	}

	return c.compile(scriptBodyBytes)
}
