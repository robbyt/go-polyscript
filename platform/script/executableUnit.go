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

	// closerOnce builds closer, which wraps the content's Close so it runs
	// at most once. closer stays nil for content without a Close method.
	closerOnce sync.Once
	closer     *onceCloser

	// lifetime is the current BindLifetime registration. Only funcs and
	// handles that undo it are kept, never the lifetime ctx itself.
	lifetimeMu sync.Mutex
	lifetime   *lifetimeBinding
}

// lifetimeBinding undoes one BindLifetime registration: stop unregisters the
// context.AfterFunc hook, and cleanup is the runtime.AddCleanup that would
// otherwise call stop once the unit's close guard is collected.
type lifetimeBinding struct {
	stop    func() bool
	cleanup runtime.Cleanup
}

// release unregisters both the hook and its cleanup, so the cleanup no
// longer keeps the hook, and through it the lifetime ctx, reachable.
func (b *lifetimeBinding) release() {
	b.cleanup.Stop()
	b.stop()
}

// contentCloser is implemented by compiled content that holds resources.
type contentCloser interface {
	Close(context.Context) error
}

// onceCloser calls a content's Close at most once. The first caller runs it
// with its own ctx; concurrent callers block in Do until it returns, and
// every caller, including later ones, gets its result.
type onceCloser struct {
	once  sync.Once
	err   error
	inner contentCloser
}

// Close closes the wrapped content on the first call and returns that
// call's result on every call.
func (c *onceCloser) Close(ctx context.Context) error {
	c.once.Do(func() { c.err = c.inner.Close(ctx) })
	return c.err
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
// The hook holds the unit only weakly, so lifetime keeps neither the unit
// nor its content in memory. A dropped unit is only known to be gone once
// it has been garbage collected: until then, the end of lifetime still
// closes the content, even if the unit is no longer reachable and a caller
// retained the content. Once the unit is collected its hook is unregistered
// and the end of lifetime no longer closes the content: a caller that
// retained the content from GetContent can keep using it and owns its
// release from then on. A caller that needs the content to outlive the unit
// should therefore not bind the unit to a lifetime that may end first. Only
// the content itself is covered: the bytecode from its GetByteCode need not
// keep the content alive, so retain the content, not just its bytecode.
// Content that holds resources should release them from its own cleanup
// when it becomes unreachable, as [os.File] does; Close releases it sooner.
//
// If lifetime is done by the time the hook is registered, BindLifetime
// closes the content before returning and returns an error that wraps
// lifetime's error and its cause, joined with any error from closing the
// content. The caller should then discard the unit. Binding again replaces
// the earlier lifetime, even when the new bind fails because its lifetime
// is done, since the content is closed then. It returns an error for a nil
// lifetime and is a no-op on a nil unit.
func (exe *ExecutableUnit) BindLifetime(lifetime context.Context) error {
	if exe == nil {
		return nil
	}
	if lifetime == nil {
		return errors.New("lifetime context is nil")
	}
	guard := exe.closeGuard()
	if guard == nil {
		if lifetime.Err() != nil {
			return lifetimeEndedError(lifetime, nil)
		}
		return nil
	}

	release := context.WithoutCancel(lifetime)
	if lifetime.Err() != nil {
		return exe.closeForEndedLifetime(lifetime, guard, release)
	}

	stop := context.AfterFunc(lifetime, closeOnLifetimeEnd(weak.Make(guard), release, exe.logger))
	// Check again now the hook is registered, so a lifetime that ends at any
	// point before BindLifetime returns is reported instead of closing the
	// content behind the caller's back. Whether or not the hook has started,
	// the guard closes the content once and makes every caller wait for that
	// close's result.
	if lifetime.Err() != nil {
		stop()
		return exe.closeForEndedLifetime(lifetime, guard, release)
	}
	// The hook cannot reach a collected guard, so unregister it then, so
	// that a dropped unit leaves nothing behind in a long-lived lifetime.
	exe.replaceLifetime(&lifetimeBinding{
		stop:    stop,
		cleanup: runtime.AddCleanup(guard, unregisterHook, stop),
	})
	return nil
}

// closeForEndedLifetime handles a lifetime that ended before BindLifetime
// could finish: it drops any earlier lifetime, closes the content through
// guard and reports both.
func (exe *ExecutableUnit) closeForEndedLifetime(
	lifetime context.Context,
	guard *onceCloser,
	release context.Context,
) error {
	exe.replaceLifetime(nil)
	return lifetimeEndedError(lifetime, guard.Close(release))
}

// closeOnLifetimeEnd returns the hook BindLifetime registers on a lifetime.
// It closes the content through guard if the unit that owns guard is still
// alive, and does nothing once it has been garbage collected.
func closeOnLifetimeEnd(guard weak.Pointer[onceCloser], release context.Context, logger *slog.Logger) func() {
	return func() {
		g := guard.Value()
		if g == nil {
			return
		}
		if err := g.Close(release); err != nil {
			logger.Warn("failed to close content when its lifetime ended", "error", err)
		}
	}
}

// unregisterHook is the cleanup BindLifetime attaches to a unit's close
// guard. It unregisters the lifetime hook, which refers to the guard only
// weakly and so cannot keep it alive.
func unregisterHook(stop func() bool) {
	stop()
}

// replaceLifetime installs binding as the unit's lifetime registration, or
// none for a nil binding, and releases the registration it replaces.
func (exe *ExecutableUnit) replaceLifetime(binding *lifetimeBinding) {
	exe.lifetimeMu.Lock()
	previous := exe.lifetime
	exe.lifetime = binding
	exe.lifetimeMu.Unlock()
	if previous != nil {
		previous.release()
	}
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

// Close releases resources held by the compiled content. Content that
// implements Close(context.Context) error is closed; for other content this
// is a no-op. It first unregisters the hook from BindLifetime, so ending the
// lifetime after Close does nothing.
//
// The content is closed at most once, using a [sync.Once]: the first close,
// whether from Close, the lifetime hook or BindLifetime, calls the content's
// Close with its own ctx, and every other Close, concurrent or later, waits
// for that call and returns its result. So only the first close's ctx bounds
// the wait. Safe on a nil unit.
func (exe *ExecutableUnit) Close(ctx context.Context) error {
	if exe == nil {
		return nil
	}
	exe.replaceLifetime(nil)
	return exe.closeContent(ctx)
}

// closeContent closes the content once, if it implements Close.
func (exe *ExecutableUnit) closeContent(ctx context.Context) error {
	if guard := exe.closeGuard(); guard != nil {
		return guard.Close(ctx)
	}
	return nil
}

// closeGuard returns the wrapper that closes the content at most once, or
// nil for content without a Close method. It is built on first use, so a
// unit not made by NewExecutableUnit gets one too.
func (exe *ExecutableUnit) closeGuard() *onceCloser {
	exe.closerOnce.Do(func() {
		if c, ok := exe.content.(contentCloser); ok {
			exe.closer = &onceCloser{inner: c}
		}
	})
	return exe.closer
}
