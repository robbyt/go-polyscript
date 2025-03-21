package script

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
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

	// Loader is responsible for loading the script content from its source (file, string, etc.).
	Loader loader.Loader

	// Content holds the compiled bytecode and source representation of the script.
	Content ExecutableContent

	// Compiler is the script language-specific compiler that was used to compile this unit.
	Compiler Compiler

	// ScriptData contains key-value pairs that can be accessed by the script at runtime.
	// This data is made available to the script during evaluation.
	ScriptData map[string]any

	// DataProvider provides runtime data for script evaluation.
	// When using the "compile once, run many times" pattern, a ContextProvider
	// is recommended to pass different data for each evaluation.
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
	loader loader.Loader,
	compiler Compiler,
	dataProvider data.Provider,
	sData map[string]any,
) (*ExecutableUnit, error) {
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

	if sData == nil {
		sData = make(map[string]any)
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
		ID:           versionID,
		CreatedAt:    time.Now(),
		Loader:       loader,
		Content:      exe,
		Compiler:     compiler,
		ScriptData:   sData,
		DataProvider: dataProvider,
		logHandler:   handler,
		logger:       logger,
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
	return ver.ScriptData
}

// GetDataProvider returns the data provider for this executable unit.
func (ver *ExecutableUnit) GetDataProvider() data.Provider {
	return ver.DataProvider
}

// WithDataProvider returns a copy of this executable unit with the specified data provider.
// This is useful for the "compile once, run many times" pattern.
func (ver *ExecutableUnit) WithDataProvider(provider data.Provider) *ExecutableUnit {
	clone := *ver
	clone.DataProvider = provider
	return &clone
}

// StoreDataInContext enriches the provided context with script and request data
// using the ExecutableUnit's configured DataProvider.
//
// This method is a replacement for the legacy BuildEvalContext method and provides
// a more flexible approach to storing data for script execution. It supports the
// "compile once, run many times with different data" pattern by using a Provider
// to manage the data.
//
// Key behaviors:
// 1. Delegates data storage to the DataProvider's AddDataToContext method
// 2. Different provider types handle data in context-appropriate ways:
//   - ContextProvider: Stores in the context under a specific key
//   - StaticProvider: Already has predefined data, doesn't support adding more
//   - CompositeProvider: Distributes data to multiple providers in a chain
//
// 3. Sends both the HTTP request and the ExecutableUnit's script data to the provider
// 4. Returns the original context if no provider is configured or if there's an error
// 5. Logs errors using the ExecutableUnit's logger
//
// Even if the provider encounters errors (e.g., can't convert a request to a map),
// all providers are designed to store whatever data they can process successfully.
// This ensures scripts always have a predictable data structure to work with.
func (ver *ExecutableUnit) StoreDataInContext(ctx context.Context, r *http.Request) context.Context {
	// If no DataProvider is configured, log an error and return original context
	if ver.DataProvider == nil {
		if ver.logger != nil {
			ver.logger.Error("Cannot store data in context: no DataProvider configured")
		}
		return ctx
	}

	// Use the DataProvider to store data
	// Note: We send both the request and script data to be handled by the provider
	// The provider will handle converting the request to a map and merging script data
	newCtx, err := ver.DataProvider.AddDataToContext(ctx, r, ver.GetScriptData())

	// If there's an error, log it but still return the context that was created
	// This allows for partial success - some data may have been stored successfully
	if err != nil {
		if ver.logger != nil {
			ver.logger.Error("Failed to add data to context using provider", "error", err)
		} else {
			defaultLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			defaultLogger.Error("Failed to add data to context using provider - logger was nil", "error", err)
		}
	}

	// Return the updated context, which may have partial data even if there was an error
	return newCtx
}
