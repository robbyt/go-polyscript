package script

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"
	"weak"

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

	// stopLifetime unregisters the hook that BindLifetime registers on the
	// lifetime ctx. Only this func is kept, never the ctx itself.
	lifetimeMu   sync.Mutex
	stopLifetime func() bool
}

// contentCloser is implemented by compiled content that holds resources.
type contentCloser interface {
	Close(context.Context) error
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

// BindLifetime ties the unit to lifetime: when lifetime is done, the unit
// closes its content as if Close had been called. Engine constructors call
// it with the ctx passed to them, or the one given to their WithLifetime
// option. Only a func that unregisters the hook is kept; lifetime itself is
// not stored. Content without a Close method holds nothing to release, so
// for it no hook is registered.
//
// The hook refers to the unit only weakly, so a long-lived lifetime does
// not keep a unit that is no longer used in memory. If such a unit is
// garbage collected while still bound and not closed, its hook is
// unregistered and its content is closed then. Call Close to release the
// content promptly instead of waiting for the garbage collector.
//
// If lifetime is already done, BindLifetime closes the content right away
// and returns an error that wraps lifetime's error and its cause, joined
// with any error from closing the content. The caller should then discard
// the unit. Binding again replaces the earlier lifetime. It returns an error
// for a nil lifetime and is a no-op on a nil unit.
func (exe *ExecutableUnit) BindLifetime(lifetime context.Context) error {
	if exe == nil {
		return nil
	}
	if lifetime == nil {
		return errors.New("lifetime context is nil")
	}
	if lifetime.Err() != nil {
		err := lifetimeEndedError(lifetime)
		if closeErr := exe.closeContent(context.WithoutCancel(lifetime)); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close content: %w", closeErr))
		}
		return err
	}
	closer, ok := exe.content.(contentCloser)
	if !ok {
		return nil
	}

	unit := weak.Make(exe)
	stop := context.AfterFunc(lifetime, func() {
		u := unit.Value()
		if u == nil {
			return
		}
		if err := u.closeContent(context.WithoutCancel(lifetime)); err != nil {
			u.logger.Warn("failed to close content when its lifetime ended", "error", err)
		}
	})
	runtime.AddCleanup(exe, closeUnreachable, unreachableUnit{
		stop:   stop,
		closer: closer,
		logger: exe.logger,
	})

	exe.lifetimeMu.Lock()
	previous := exe.stopLifetime
	exe.stopLifetime = stop
	exe.lifetimeMu.Unlock()
	if previous != nil {
		previous()
	}
	return nil
}

// unreachableUnit is what the cleanup registered by BindLifetime needs to
// release a unit's content after the unit is garbage collected. It must not
// refer to the unit itself.
type unreachableUnit struct {
	stop   func() bool
	closer contentCloser
	logger *slog.Logger
}

// closeUnreachable unregisters the lifetime hook of a garbage-collected
// unit and closes its content, unless Close, the hook or a later
// BindLifetime already stopped that hook. The close runs on its own
// goroutine so a slow one does not hold up other cleanups.
func closeUnreachable(u unreachableUnit) {
	if !u.stop() {
		return
	}
	go func() {
		if err := u.closer.Close(context.Background()); err != nil {
			u.logger.Warn("failed to close content of an unreachable unit", "error", err)
		}
	}()
}

// lifetimeEndedError reports that lifetime was done before the unit could be
// bound to it. It wraps both ctx.Err() and, when it differs, the cause, so
// errors.Is matches either.
func lifetimeEndedError(lifetime context.Context) error {
	err := lifetime.Err()
	if cause := context.Cause(lifetime); cause != nil && !errors.Is(err, cause) {
		return fmt.Errorf("lifetime context ended before construction finished: %w (cause: %w)", err, cause)
	}
	return fmt.Errorf("lifetime context ended before construction finished: %w", err)
}

// Close releases resources held by the compiled content. Content that
// implements Close(context.Context) error is closed; for other content this
// is a no-op. It first unregisters the hook from BindLifetime, so ending the
// lifetime after Close does nothing. ctx bounds how long Close waits for the
// content, as documented by the content's own Close. Safe to call more than
// once and on a nil unit.
func (exe *ExecutableUnit) Close(ctx context.Context) error {
	if exe == nil {
		return nil
	}
	exe.lifetimeMu.Lock()
	stop := exe.stopLifetime
	exe.stopLifetime = nil
	exe.lifetimeMu.Unlock()
	if stop != nil {
		stop()
	}
	return exe.closeContent(ctx)
}

// closeContent closes the content if it implements Close.
func (exe *ExecutableUnit) closeContent(ctx context.Context) error {
	if closer, ok := exe.content.(contentCloser); ok {
		return closer.Close(ctx)
	}
	return nil
}
