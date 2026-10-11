package compiler

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/robbyt/go-polyscript/engines/extism/adapters"
	engineTypes "github.com/robbyt/go-polyscript/engines/types"
)

// ErrExecutableClosed is returned when a closed Executable's compiled plugin
// is used, e.g. by an Eval after the evaluator's Close.
var ErrExecutableClosed = errors.New("executable is closed")

// Executable implements script.ExecutableContent for Extism WASM modules.
// It owns the compiled plugin (and its wazero runtime) until it is released
// by Close.
type Executable struct {
	scriptBytes []byte
	byteCode    adapters.CompiledPlugin
	entryPoint  string

	// mu guards the fields below. inFlight counts running WithPlugin calls.
	// closing is set by the first Close; from then on WithPlugin fails. The
	// plugin is released once closing is set and inFlight is zero, by
	// whichever of Close or the last WithPlugin sees that first;
	// releaseStarted makes sure only one does. released is closed after the
	// release, and releaseErr, written before that, holds its error.
	mu             sync.Mutex
	inFlight       int
	closing        bool
	releaseStarted bool
	released       chan struct{}
	releaseErr     error
}

// NewExecutable creates a new Executable instance
func NewExecutable(
	scriptBytes []byte,
	byteCode adapters.CompiledPlugin,
	entryPoint string,
) *Executable {
	if len(scriptBytes) == 0 || byteCode == nil || entryPoint == "" {
		return nil
	}
	return &Executable{
		scriptBytes: scriptBytes,
		byteCode:    byteCode,
		entryPoint:  entryPoint,
		released:    make(chan struct{}),
	}
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
// concurrent Close; use WithPlugin to run code against it safely.
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
	release := e.claimReleaseLocked()
	e.mu.Unlock()
	if release {
		e.release(context.WithoutCancel(ctx))
	}
}

// claimReleaseLocked reports whether the caller should release the plugin
// now: Close has been called, nothing is in flight, and no one else has
// claimed the release. e.mu must be held.
func (e *Executable) claimReleaseLocked() bool {
	if !e.closing || e.inFlight > 0 || e.releaseStarted {
		return false
	}
	e.releaseStarted = true
	return true
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
	var err error
	if e.byteCode != nil {
		err = e.byteCode.Close(ctx)
	}
	e.mu.Lock()
	e.releaseErr = err
	ch := e.releasedLocked()
	e.mu.Unlock()
	close(ch)
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
// the release failed. Safe on a nil Executable.
func (e *Executable) Close(ctx context.Context) error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	released := e.releasedLocked()
	e.closing = true
	release := e.claimReleaseLocked()
	e.mu.Unlock()

	if release {
		e.release(context.WithoutCancel(ctx))
	}
	select {
	case <-released:
		return e.loadReleaseErr()
	case <-ctx.Done():
	}
	// ctx is done, but only in-flight calls may make Close give up. With
	// none left, the release is claimed and runs without cancellation, so
	// wait for it.
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
