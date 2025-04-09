package compiler

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/machines/starlark/compiler/internal/compile"
)

type Compiler struct {
	globals    []string
	logHandler slog.Handler
	logger     *slog.Logger
}

// NewCompiler creates a new Starlark-specific Compiler instance with the provided options.
// Global variables are used during script parsing to validate global name usage.
func NewCompiler(opts ...FunctionalOption) (*Compiler, error) {
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

func (c *Compiler) compile(scriptBodyBytes []byte) (*executable, error) {
	logger := c.logger.WithGroup("compile")
	if len(scriptBodyBytes) == 0 {
		logger.Error("Compile called with nil script")
		return nil, ErrContentNil
	}

	logger.Debug("Starting Starlark compilation", "scriptLength", len(scriptBodyBytes))

	// Compile the script with globals
	program, err := compile.CompileWithEmptyGlobals(scriptBodyBytes, c.globals)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	if program == nil {
		return nil, ErrBytecodeNil
	}

	// Create executable with the compiled program
	starlarkExec := newExecutable(scriptBodyBytes, program)
	if starlarkExec == nil {
		return nil, ErrExecCreationFailed
	}

	logger.Debug("Starlark compilation completed")
	return starlarkExec, nil
}
