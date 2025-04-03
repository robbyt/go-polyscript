package script

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/internal/helpers"
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
// Static script data (sData) is automatically combined with the runtime data provider using a CompositeProvider.
func NewExecutableUnit(
	handler slog.Handler,
	versionID string,
	scriptLoader loader.Loader,
	compiler Compiler,
	dataProvider data.Provider,
	sData map[string]any,
) (*ExecutableUnit, error) {
	handler, logger := helpers.SetupLogger(handler, "script", "ExecutableUnit")

	if compiler == nil {
		return nil, errors.New("compiler is nil")
	}

	if sData == nil {
		sData = make(map[string]any)
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

	// Create a StaticProvider for the compile-time script data
	staticProvider := data.NewStaticProvider(sData)

	// Create a CompositeProvider that combines the static compile-time data with
	// the runtime data from the passed-in provider. The ordering is important:
	// runtime data should override compile-time data for any duplicate keys.
	var combinedProvider data.Provider
	if dataProvider != nil {
		combinedProvider = data.NewCompositeProvider(staticProvider, dataProvider)
	} else {
		// If no runtime data provider is specified, just use the static provider
		combinedProvider = staticProvider
	}

	return &ExecutableUnit{
		ID:           versionID,
		CreatedAt:    time.Now(),
		ScriptLoader: scriptLoader,
		Content:      exe,
		Compiler:     compiler,
		DataProvider: combinedProvider,
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

// GetScriptData returns the script data associated with this version.
//
// Deprecated: This method is maintained for backwards compatibility but now retrieves
// data from the DataProvider. It may be removed in a future version.
// Prefer using GetDataProvider() directly to access script data.
func (exe *ExecutableUnit) GetScriptData() map[string]any {
	// For backwards compatibility, attempt to extract static data if it's a composite provider
	if exe.DataProvider == nil {
		return nil
	}

	// If we can get the data synchronously from the provider, return it
	// This will work for StaticProvider and return the full merged data for CompositeProvider
	data, err := exe.DataProvider.GetData(context.Background())
	if err != nil {
		// If there's an error, just return an empty map
		return make(map[string]any)
	}
	return data
}

// GetDataProvider returns the data provider for this executable unit.
func (exe *ExecutableUnit) GetDataProvider() data.Provider {
	return exe.DataProvider
}
