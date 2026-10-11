package compiler

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/robbyt/go-polyscript/engines/extism/adapters"
	engineTypes "github.com/robbyt/go-polyscript/engines/types"
)

// ErrExecutableClosed is returned when a closed Executable's compiled plugin
// is used, e.g. by an Eval after the evaluator's Close.
var ErrExecutableClosed = errors.New("executable is closed")

// Executable implements script.ExecutableContent for Extism WASM modules.
// It owns the compiled plugin (and its wazero runtime) until it is released
// by Close. Like an [os.File], an Executable from NewExecutable that becomes
// unreachable without being closed releases its plugin when it is garbage
// collected; Close releases it sooner. Retaining an Executable, for example
// from an ExecutableUnit's GetContent, keeps the garbage collector from
// releasing it, and whoever retains it owns its release; the unit can still
// close it through its own Close or the end of its lifetime until the unit
// has been garbage collected. Retaining only the plugin from
// GetExtismByteCode does not keep the Executable, or the plugin, open.
type Executable struct {
	scriptBytes []byte
	byteCode    adapters.CompiledPlugin
	entryPoint  string

	// rel releases the plugin at most once, for both Close and the cleanup
	// NewExecutable registers. It is built on first use for an Executable
	// not made by NewExecutable.
	rel *pluginRelease

	// releaseOnce runs the release exactly once. The plugin is released
	// once closing is set and inFlight is zero, by whichever of Close or the
	// last WithPlugin sees that first; a caller that arrives while the
	// release runs waits in Do until it finishes.
	releaseOnce sync.Once

	// mu guards the fields below. inFlight counts running WithPlugin calls.
	// closing is set by the first Close; from then on WithPlugin fails.
	// released is closed after the release, and releaseErr, written before
	// that, holds its error.
	mu         sync.Mutex
	inFlight   int
	closing    bool
	released   chan struct{}
	releaseErr error
}

// pluginRelease closes a compiled plugin at most once. It does not refer to
// the Executable, so the cleanup that releases an unreachable Executable can
// hold it without keeping that Executable alive.
type pluginRelease struct {
	once   sync.Once
	plugin adapters.CompiledPlugin
	err    error
}

// release closes the plugin on the first call and returns that close's
// error on every call. Concurrent callers wait for the first to finish.
func (r *pluginRelease) release(ctx context.Context) error {
	r.closeOnce(ctx)
	return r.err
}

// closeOnce closes the plugin on the first call and stores its error.
func (r *pluginRelease) closeOnce(ctx context.Context) {
	r.once.Do(func() {
		if r.plugin != nil {
			r.err = r.plugin.Close(ctx)
		}
	})
}

// releaseUnreachable is the cleanup NewExecutable registers. It releases
// the plugin of an Executable that was garbage collected without being
// released; for one that was, it does nothing. Nothing is left to report a
// release error to, so it is kept only in r.
func releaseUnreachable(r *pluginRelease) {
	r.closeOnce(context.Background())
}

// NewExecutable creates a new Executable instance. It returns nil if
// scriptBytes is empty, byteCode is nil or entryPoint is empty. If the
// Executable becomes unreachable without being closed, its plugin is
// released when it is garbage collected.
func NewExecutable(
	scriptBytes []byte,
	byteCode adapters.CompiledPlugin,
	entryPoint string,
) *Executable {
	if len(scriptBytes) == 0 || byteCode == nil || entryPoint == "" {
		return nil
	}
	e := &Executable{
		scriptBytes: scriptBytes,
		byteCode:    byteCode,
		entryPoint:  entryPoint,
		rel:         &pluginRelease{plugin: byteCode},
		released:    make(chan struct{}),
	}
	runtime.AddCleanup(e, releaseUnreachable, e.rel)
	return e
}

// GetSource returns the original script content
func (e *Executable) GetSource() string {
	return string(e.scriptBytes)
}

// GetByteCode returns the compiled plugin as a generic interface, or nil
// once Close has been called.
func (e *Executable) GetByteCode() any {
	return e.GetExtismByteCode()
}

// GetExtismByteCode returns the compiled plugin with its proper type, or nil
// once Close has been called. The returned plugin is not protected against a
// concurrent Close, and does not keep the Executable from being garbage
// collected and releasing it; use WithPlugin to run code against it safely.
func (e *Executable) GetExtismByteCode() adapters.CompiledPlugin {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closing {
		return nil
	}
	return e.byteCode
}

// WithPlugin calls fn with the compiled plugin and keeps the plugin open
// until fn returns: a concurrent Close does not release it before then.
// Concurrent WithPlugin calls do not block each other. It returns
// ErrExecutableClosed without calling fn once Close has been called, even
// while that Close is still waiting, ErrBytecodeNil if there is no plugin,
// and otherwise fn's error. If Close is waiting, or gave up waiting, when
// the last WithPlugin call returns, that call releases the plugin with ctx's
// values but not its cancellation. fn must not call Close on e with a ctx
// that is never done, since Close would wait for fn.
func (e *Executable) WithPlugin(ctx context.Context, fn func(plugin adapters.CompiledPlugin) error) error {
	e.mu.Lock()
	if e.closing {
		e.mu.Unlock()
		return ErrExecutableClosed
	}
	if e.byteCode == nil {
		e.mu.Unlock()
		return ErrBytecodeNil
	}
	e.inFlight++
	plugin := e.byteCode
	e.mu.Unlock()

	defer e.done(ctx)
	return fn(plugin)
}

// done ends a WithPlugin call. The last call to end after Close has been
// called releases the plugin, without ctx's cancellation.
func (e *Executable) done(ctx context.Context) {
	e.mu.Lock()
	e.inFlight--
	ready := e.readyToReleaseLocked()
	e.mu.Unlock()
	if ready {
		e.releaseOnce.Do(func() { e.release(context.WithoutCancel(ctx)) })
	}
}

// readyToReleaseLocked reports whether the plugin can be released now:
// Close has been called and nothing is in flight. releaseOnce makes sure
// only one caller releases it. e.mu must be held.
func (e *Executable) readyToReleaseLocked() bool {
	return e.closing && e.inFlight == 0
}

// releasedLocked returns the channel closed once the plugin is released,
// creating it for an Executable not built by NewExecutable. e.mu must be
// held.
func (e *Executable) releasedLocked() chan struct{} {
	if e.released == nil {
		e.released = make(chan struct{})
	}
	return e.released
}

// release closes the plugin and then the released channel. Callers pass a
// ctx without cancellation, so the release is never cut short.
func (e *Executable) release(ctx context.Context) {
	e.mu.Lock()
	rel := e.relLocked()
	e.mu.Unlock()
	err := rel.release(ctx)
	e.mu.Lock()
	e.releaseErr = err
	ch := e.releasedLocked()
	e.mu.Unlock()
	close(ch)
}

// relLocked returns the plugin's release, creating it for an Executable not
// built by NewExecutable. e.mu must be held.
func (e *Executable) relLocked() *pluginRelease {
	if e.rel == nil {
		e.rel = &pluginRelease{plugin: e.byteCode}
	}
	return e.rel
}

// IsClosed reports whether the plugin has been released, not merely that
// Close has been called. It stays false while a Close waits for in-flight
// WithPlugin calls, or after a Close that gave up waiting, until the last of
// those calls returns and the release finishes.
func (e *Executable) IsClosed() bool {
	e.mu.Lock()
	ch := e.releasedLocked()
	e.mu.Unlock()
	return isClosed(ch)
}

// EngineType returns the Extism engine type
func (e *Executable) EngineType() engineTypes.Type {
	return engineTypes.Extism
}

// GetEntryPoint returns the name of the entry point function
func (e *Executable) GetEntryPoint() string {
	return e.entryPoint
}

// Close releases the compiled plugin and its wazero runtime. As soon as it
// is called, new WithPlugin calls fail with ErrExecutableClosed. With
// nothing in flight it releases the plugin, or waits for a release already
// under way, and returns the release error, even if ctx is already done.
// Otherwise it waits for the in-flight WithPlugin calls (i.e. running Evals)
// until ctx is done: if they finish first, the last one releases the plugin
// and Close returns the release error; if ctx is done first, Close returns
// ctx's error and the plugin is released when the last of them returns. The
// release never uses ctx's cancellation. Close is safe to call more than
// once and concurrently: the plugin is released once, and every Close that
// sees the release finish, including one called after it, returns its
// error. Call Close again after one that gave up waiting to learn whether
// the release failed. Close releases the plugin sooner than the garbage
// collector, which releases an unclosed Executable once it is unreachable.
// Safe on a nil Executable.
func (e *Executable) Close(ctx context.Context) error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	released := e.releasedLocked()
	e.closing = true
	ready := e.readyToReleaseLocked()
	e.mu.Unlock()

	if ready {
		e.releaseOnce.Do(func() { e.release(context.WithoutCancel(ctx)) })
	}
	select {
	case <-released:
		return e.loadReleaseErr()
	case <-ctx.Done():
	}
	// ctx is done, but only in-flight calls may make Close give up. With
	// none left, the release is under way without cancellation, so wait for
	// it.
	if e.hasInFlight() {
		return fmt.Errorf("extism: close stopped waiting for in-flight evals: %w", ctx.Err())
	}
	<-released
	return e.loadReleaseErr()
}

// hasInFlight reports whether any WithPlugin call is still running.
func (e *Executable) hasInFlight() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.inFlight > 0
}

// loadReleaseErr returns the error from releasing the plugin.
func (e *Executable) loadReleaseErr() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.releaseErr
}

// isClosed reports whether ch is closed.
func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
