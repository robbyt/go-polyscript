package script

import (
	"context"
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

// contentCloser is implemented by content that holds resources.
type contentCloser interface {
	Close(context.Context) error
}

// lifetimeWatcher is implemented by content that releases its resources
// when a lifetime ends. Like [io.WriterTo], it is optional: BindLifetime
// passes the lifetime's Done channel to content that implements it.
type lifetimeWatcher interface {
	WatchLifetime(done <-chan struct{})
}

// NewExecutableUnit creates a new ExecutableUnit from the provided loader and compiler.
// The dataProvider parameter provides runtime data for script evaluation.
// A nil handler is permitted and inherits from slog.Default.
// The supplied ctx flows into the loader (for IO cancellation) and the
// compiler (for parse/compile cancellation); it is not retained on the
// returned unit.
func NewExecutableUnit(
	ctx context.Context,
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

	reader, err := scriptLoader.GetReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get reader from loader: %w", err)
	}

	exe, err := compiler.Compile(ctx, reader)
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

// BindLifetime ties the content's resources to lifetime. Content that
// implements WatchLifetime(done <-chan struct{}) is given lifetime.Done(),
// and closes itself once that channel is closed. Only the channel is passed
// on; lifetime itself is not stored. Engine constructors call BindLifetime
// once, with the ctx passed to them or the one given to their WithLifetime
// option. It must not be called again.
//
// After handing over the channel, BindLifetime checks lifetime once more. If
// lifetime is already done, it closes the content and returns an error that
// wraps lifetime's error and, when it differs, its cause, joined with any
// error from closing the content; the caller should then discard the unit.
// Content that does not watch its lifetime gets the same error. A lifetime
// that ends after that check, even before the constructor returns, closes
// the content just as if it ended later; only a lifetime already over at the
// check fails construction. It returns an error for a nil lifetime and is a
// no-op on a nil unit.
func (exe *ExecutableUnit) BindLifetime(lifetime context.Context) error {
	if exe == nil {
		return nil
	}
	if lifetime == nil {
		return errors.New("lifetime context is nil")
	}
	if w, ok := exe.content.(lifetimeWatcher); ok {
		if done := lifetime.Done(); done != nil {
			w.WatchLifetime(done)
		}
	}
	if lifetime.Err() == nil {
		return nil
	}
	return lifetimeEndedError(lifetime, exe.Close(context.WithoutCancel(lifetime)))
}

// lifetimeEndedError reports that lifetime was done before the unit could be
// bound to it. It wraps both ctx.Err() and, when it differs, the cause, so
// errors.Is matches either, joined with closeErr from closing the content.
func lifetimeEndedError(lifetime context.Context, closeErr error) error {
	err := lifetime.Err()
	if cause := context.Cause(lifetime); cause != nil && !errors.Is(err, cause) {
		err = fmt.Errorf("lifetime context ended before construction finished: %w (cause: %w)", err, cause)
	} else {
		err = fmt.Errorf("lifetime context ended before construction finished: %w", err)
	}
	if closeErr != nil {
		err = errors.Join(err, fmt.Errorf("failed to close content: %w", closeErr))
	}
	return err
}

// Close releases resources held by the compiled content by calling its
// Close, for content that implements Close(context.Context) error; for other
// content it is a no-op. The content makes repeated and concurrent calls
// safe, and ctx bounds only this caller's wait (see [ExecutableContent]).
// Safe on a nil unit.
func (exe *ExecutableUnit) Close(ctx context.Context) error {
	if exe == nil {
		return nil
	}
	if c, ok := exe.content.(contentCloser); ok {
		return c.Close(ctx)
	}
	return nil
}
