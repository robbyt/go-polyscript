package compiler

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/machines/risor/compiler/internal/compile"
)

type Compiler struct {
	globals    []string
	logHandler slog.Handler
	logger     *slog.Logger
}

// NewCompiler creates a new Risor-specific Compiler instance with the provided options.
// Global variables are used for initial script parsing while building the executable bytecode.
func NewCompiler(opts ...FunctionalOption) (*Compiler, error) {
	// Initialize config with defaults
	cfg := &Options{}
	ApplyDefaults(cfg)

	// Apply all options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("error applying compiler option: %w", err)
		}
	}

	// Validate the configuration
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid compiler configuration: %w", err)
	}

	var handler slog.Handler
	var logger *slog.Logger

	// Set up logging based on provided options
	if cfg.Logger != nil {
		// User provided a custom logger
		logger = cfg.Logger
		handler = logger.Handler()
	} else {
		// User provided a handler or we're using the default
		handler, logger = helpers.SetupLogger(cfg.LogHandler, "risor", "Compiler")
	}

	return &Compiler{
		globals:    cfg.Globals,
		logHandler: handler,
		logger:     logger,
	}, nil
}

func (c *Compiler) String() string {
	return "risor.Compiler"
}

// Compile turns the provided script content into runnable bytecode.
func (c *Compiler) Compile(scriptLoader io.ReadCloser) (script.ExecutableContent, error) {
	if scriptLoader == nil {
		return nil, ErrContentNil
	}

	scriptBodyBytes, err := io.ReadAll(scriptLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to read script: %w", err)
	}

	err = scriptLoader.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close reader: %w", err)
	}

	return c.compile(scriptBodyBytes)
}

func (c *Compiler) compile(scriptBodyBytes []byte) (*executable, error) {
	logger := c.logger.WithGroup("compile")
	if len(scriptBodyBytes) == 0 {
		return nil, ErrContentNil
	}
	scriptContent := string(scriptBodyBytes)

	// Check for empty script
	trimmedScript := strings.TrimSpace(scriptContent)
	if trimmedScript == "" {
		logger.Warn("Empty script content")
		return nil, ErrNoInstructions
	}

	// Check for comment-only script
	isCommentOnly := true
	for line := range strings.SplitSeq(trimmedScript, "\n") {
		if trimmedLine := strings.TrimSpace(line); trimmedLine != "" &&
			!strings.HasPrefix(trimmedLine, "#") {
			// Found a non-comment line, so we can stop checking lines because there's some real code here!
			isCommentOnly = false
			break
		}
	}
	if isCommentOnly {
		logger.Warn("Script contains only comments")
		return nil, ErrNoInstructions
	}

	logger.Debug("Starting validation", "script", scriptContent, "globals", c.globals)

	bc, err := compile.CompileWithGlobals(&scriptContent, c.globals)
	if err != nil {
		logger.Warn("Compilation failed", "error", err)
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	if bc == nil {
		logger.Error("Compilation returned nil bytecode")
		return nil, ErrBytecodeNil
	}

	instructionCount := bc.InstructionCount()
	logger.Debug("Compilation successful", "instructionCount", instructionCount)
	if instructionCount < 1 {
		logger.Warn("Bytecode has zero instructions")
		return nil, ErrNoInstructions
	}

	risorExec := newExecutable(scriptBodyBytes, bc)
	if risorExec == nil {
		logger.Warn("Failed to create Executable from bytecode")
		return nil, ErrExecCreationFailed
	}

	logger.Debug("Validation completed")
	return risorExec, nil
}
