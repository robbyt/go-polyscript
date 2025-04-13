package script

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/robbyt/go-polyscript/abstract/data"
	"github.com/robbyt/go-polyscript/abstract/script/loader"
	machineTypes "github.com/robbyt/go-polyscript/engines/types"
	"github.com/robbyt/go-polyscript/internal/helpers"
)

const checksumLength = 12

// ExecutableUnit represents a specific version of a script, including its content and creation time.
// It holds the compiled script content and provides access to evaluation facilities.
type ExecutableUnit struct {
	// ID is a unique identifier for this executable unit, typically derived from a hash of the script content.
	ID string

	// CreatedAt records when this executable unit was instantiated.
	CreatedAt time.Time

	// ScriptLoader loads the script content to local memory from various places (file, string, etc.).
	ScriptLoader loader.Loader

	// Compiler is the script language-specific compiler that was used to compile this unit.
	Compiler Compiler

	// Content holds the compiled bytecode and source representation of the script.
	Content ExecutableContent

	// DataProvider provides access to both static compile-time data and variable runtime data
	// during script evaluation. Enabling the "compile once, run many times" design.
	// When created with NewExecutableUnit, this is typically a CompositeProvider containing
	// a StaticProvider (for compile-time data) and another provider (for runtime data).
	DataProvider data.Provider

	// Logging components
	logHandler slog.Handler
	logger     *slog.Logger
}

// NewExecutableUnit creates a new ExecutableUnit from the provided loader and compiler.
// The dataProvider parameter provides runtime data for script evaluation.
func NewExecutableUnit(
	handler slog.Handler,
	versionID string,
	scriptLoader loader.Loader,
	compiler Compiler,
	dataProvider data.Provider,
) (*ExecutableUnit, error) {
	handler, logger := helpers.SetupLogger(handler, "script", "ExecutableUnit")

	if compiler == nil {
		return nil, errors.New("compiler is nil")
	}

	reader, err := scriptLoader.GetReader()
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

	return &ExecutableUnit{
		ID:           versionID,
		CreatedAt:    time.Now(),
		ScriptLoader: scriptLoader,
		Content:      exe,
		Compiler:     compiler,
		DataProvider: dataProvider,
		logHandler:   handler,
		logger:       logger.With("ID", versionID),
	}, nil
}

func (exe *ExecutableUnit) String() string {
	return fmt.Sprintf("ExecutableUnit{ID: %s, CreatedAt: %s, Compiler: %s, Loader: %s}",
		exe.ID, exe.CreatedAt, exe.Compiler, exe.ScriptLoader)
}

// GetID returns the unique identifier (version number, or name) for this script version.
func (exe *ExecutableUnit) GetID() string {
	return exe.ID
}

// GetContent returns the validated & compiled script content as ExecutableContent
func (exe *ExecutableUnit) GetContent() ExecutableContent {
	return exe.Content
}

// CreatedAt returns the timestamp when the version was created.
func (exe *ExecutableUnit) GetCreatedAt() time.Time {
	return exe.CreatedAt
}

// GetMachineType returns the machine type this script is intended to run on.
func (exe *ExecutableUnit) GetMachineType() machineTypes.Type {
	return exe.Content.GetMachineType()
}

// GetCompiler returns the compiler used to validate the script and convert it into runnable bytecode.
func (exe *ExecutableUnit) GetCompiler() Compiler {
	return exe.Compiler
}

// GetLoader returns the loader used to load the script.
func (exe *ExecutableUnit) GetLoader() loader.Loader {
	return exe.ScriptLoader
}

// GetDataProvider returns the data provider for this executable unit.
func (exe *ExecutableUnit) GetDataProvider() data.Provider {
	return exe.DataProvider
}
