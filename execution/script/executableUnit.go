package script

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data/helpers"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	machineTypes "github.com/robbyt/go-polyscript/machines/types"
)

const checksumLength = 12

// ExecutableUnit represents a specific version of a script, including its content and creation time.
// It holds the compiled script content and provides access to evaluation facilities.
type ExecutableUnit struct {
	// ID is a unique identifier for this executable unit, typically derived from a hash of the script content.
	ID string

	// CreatedAt records when this executable unit was instantiated.
	CreatedAt time.Time

	// Loader is responsible for loading the script content from its source (file, string, etc.).
	Loader loader.Loader

	// Content holds the compiled bytecode and source representation of the script.
	Content ExecutableContent

	// Compiler is the script language-specific compiler that was used to compile this unit.
	Compiler Compiler

	// ScriptData contains key-value pairs that can be accessed by the script at runtime.
	// This data is made available to the script during evaluation.
	ScriptData map[string]any

	// Logging components
	logHandler slog.Handler
	logger     *slog.Logger
}

// NewExecutableUnit creates a new ExecutableUnit from the provided loader and compiler.
// The EvalContext parameter allows passing variables from Go into the VM at evaluation time.
func NewExecutableUnit(handler slog.Handler, versionID string, loader loader.Loader, compiler Compiler, sData map[string]any) (*ExecutableUnit, error) {
	if handler == nil {
		defaultHandler := slog.NewTextHandler(os.Stdout, nil)
		handler = defaultHandler.WithGroup("script")
		// Create a logger from the handler rather than using slog directly
		defaultLogger := slog.New(handler)
		defaultLogger.Warn("Handler is nil, using the default logger configuration.")
	}

	if compiler == nil {
		return nil, errors.New("compiler is nil")
	}

	reader, err := loader.GetReader()
	if err != nil {
		return nil, fmt.Errorf("failed to get reader from loader: %w", err)
	}

	exe, err := compiler.Compile(reader)
	if err != nil {
		return nil, fmt.Errorf("compiler failed: %w", err)
	}

	if versionID == "" {
		versionID = helpers.SHA256(exe.GetSource())
		if len(versionID) > checksumLength {
			versionID = versionID[:checksumLength]
		}
	}

	logger := slog.New(handler.WithGroup("ExecutableUnit"))
	logger = logger.With("ID", versionID)

	return &ExecutableUnit{
		ID:         versionID,
		CreatedAt:  time.Now(),
		Loader:     loader,
		Content:    exe,
		Compiler:   compiler,
		ScriptData: sData,
		logHandler: handler,
		logger:     logger,
	}, nil
}

func (ver *ExecutableUnit) String() string {
	return fmt.Sprintf("ExecutableUnit{ID: %s, CreatedAt: %s, Compiler: %s, Loader: %s}",
		ver.ID, ver.CreatedAt, ver.Compiler, ver.Loader)
}

// GetID returns the unique identifier (version number, or name) for this script version.
func (ver *ExecutableUnit) GetID() string {
	return ver.ID
}

// GetContent returns the validated & compiled script content as ExecutableContent
func (ver *ExecutableUnit) GetContent() ExecutableContent {
	return ver.Content
}

// CreatedAt returns the timestamp when the version was created.
func (ver *ExecutableUnit) GetCreatedAt() time.Time {
	return ver.CreatedAt
}

// GetMachineType returns the machine type this script is intended to run on.
func (ver *ExecutableUnit) GetMachineType() machineTypes.Type {
	return ver.Content.GetMachineType()
}

// GetCompiler returns the compiler used to validate the script and convert it into runnable bytecode.
func (ver *ExecutableUnit) GetCompiler() Compiler {
	return ver.Compiler
}

// GetLoader returns the loader used to load the script.
func (ver *ExecutableUnit) GetLoader() loader.Loader {
	return ver.Loader
}

// GetScriptData returns the script data associated with this version.
func (ver *ExecutableUnit) GetScriptData() map[string]any {
	if ver.ScriptData != nil {
		return ver.ScriptData
	}
	return make(map[string]any)
}

// BuildEvalContext builds and returns the a context object associated with this version. It packs
// the script data and request data into the context, accessible with a call to ctx.Value(constants.EvalDataKey).
func (ver *ExecutableUnit) BuildEvalContext(ctx context.Context, r *http.Request) context.Context {
	evalData := make(map[string]any)
	evalData[constants.ScriptData] = ver.GetScriptData()

	rMap, err := helpers.RequestToMap(r)
	if err != nil {
		// Use the configured logger to log the error
		if ver.logger != nil {
			ver.logger.Error("Failed to convert request to map", "error", err)
		} else {
			// Fallback to default logger as a last resort
			defaultLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			defaultLogger.Error("Failed to convert request to map - logger was nil", "error", err)
		}
		rMap = make(map[string]any)
	}
	evalData[constants.Request] = rMap

	//nolint:staticcheck // Temporarily ignoring the "string as context key" warning until type system is fixed
	return context.WithValue(ctx, constants.EvalData, evalData)
}
