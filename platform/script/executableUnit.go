package script

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	engineTypes "github.com/robbyt/go-polyscript/engines/types"
	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

const checksumLength = 12

// ExecutableUnit represents a specific version of a script, including its content and creation time.
// It holds the compiled script content and provides access to evaluation facilities.
// Fields are unexported; construct via NewExecutableUnit and read via the getter methods.
type ExecutableUnit struct {
	id           string
	createdAt    time.Time
	scriptLoader loader.Loader
	compiler     Compiler
	content      ExecutableContent
	dataProvider data.Provider

	logHandler slog.Handler
	logger     *slog.Logger
}

// NewExecutableUnit creates a new ExecutableUnit from the provided loader and compiler.
// The dataProvider parameter provides runtime data for script evaluation.
// A nil handler is permitted and inherits from slog.Default.
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

	if scriptLoader == nil {
		return nil, errors.New("scriptLoader is nil")
	}

	reader, err := scriptLoader.GetReader()
	if err != nil {
		return nil, fmt.Errorf("failed to get reader from loader: %w", err)
	}

	exe, err := compiler.Compile(reader)
	if err != nil {
		return nil, fmt.Errorf("compiler failed: %w", err)
	}

	if exe == nil {
		return nil, errors.New("compiler returned nil content")
	}

	if versionID == "" {
		versionID = helpers.SHA256(exe.GetSource())
		if len(versionID) > checksumLength {
			versionID = versionID[:checksumLength]
		}
	}

	return &ExecutableUnit{
		id:           versionID,
		createdAt:    time.Now(),
		scriptLoader: scriptLoader,
		content:      exe,
		compiler:     compiler,
		dataProvider: dataProvider,
		logHandler:   handler,
		logger:       logger.With("ID", versionID),
	}, nil
}

func (exe *ExecutableUnit) String() string {
	return fmt.Sprintf("ExecutableUnit{ID: %s, CreatedAt: %s, Compiler: %s, Loader: %s}",
		exe.id, exe.createdAt, exe.compiler, exe.scriptLoader)
}

// GetID returns the unique identifier (version number, or name) for this script version.
func (exe *ExecutableUnit) GetID() string {
	return exe.id
}

// GetContent returns the validated & compiled script content as ExecutableContent
func (exe *ExecutableUnit) GetContent() ExecutableContent {
	return exe.content
}

// GetCreatedAt returns the timestamp when the version was created.
func (exe *ExecutableUnit) GetCreatedAt() time.Time {
	return exe.createdAt
}

// EngineType returns the engine type this script is intended to run on.
func (exe *ExecutableUnit) EngineType() engineTypes.Type {
	return exe.content.EngineType()
}

// GetCompiler returns the compiler used to validate the script and convert it into runnable bytecode.
func (exe *ExecutableUnit) GetCompiler() Compiler {
	return exe.compiler
}

// GetLoader returns the loader used to load the script.
func (exe *ExecutableUnit) GetLoader() loader.Loader {
	return exe.scriptLoader
}

// GetDataProvider returns the data provider for this executable unit.
func (exe *ExecutableUnit) GetDataProvider() data.Provider {
	return exe.dataProvider
}
