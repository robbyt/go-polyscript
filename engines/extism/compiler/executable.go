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
// It owns the compiled plugin (and its wazero runtime) until Close, or the
// end of the lifetime given to WatchLifetime, releases it. Nothing releases
// it when it is garbage collected: like an [os/exec.Cmd] that needs Wait,
// an Executable that is neither closed nor watching a lifetime that ends
// keeps its runtime.
type Executable struct {
	scriptBytes []byte
	byteCode    adapters.CompiledPlugin
	entryPoint  string

	// watchOnce makes WatchLifetime start at most one watcher.
	watchOnce sync.Once

	// releaseOnce runs the release exactly once, after closing is set: in a
	// Close (or the watcher) that finds nothing in flight, or in the last
	// WithPlugin call to end, which stays counted in inFlight until the
	// release finishes. A caller that arrives while the release runs waits
	// in Do until it finishes.
	releaseOnce sync.Once

	// mu guards the fields below. inFlight counts running WithPlugin calls,
	// including the last one while it releases the plugin.
	// closing is set by the first Close; from then on WithPlugin fails, as
	// it does once lifetimeDone is closed. released is closed after the
	// release, and releaseErr, written before that, holds its error.
	mu           sync.Mutex
	inFlight     int
	closing      bool
	lifetimeDone <-chan struct{}
	released     chan struct{}
	releaseErr   error
}

// NewExecutable creates a new Executable instance. It returns nil if
// scriptBytes is empty, byteCode is nil or entryPoint is empty.
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

// WatchLifetime makes the Executable close itself once done is closed, as
// Close does but without waiting: the plugin is released then, or when the
// last in-flight WithPlugin call returns. WithPlugin fails with
// ErrExecutableClosed as soon as done is closed, even before that close
// starts. It keeps only the channel, never a ctx, and starts one goroutine,
// which exits once done is closed or the plugin is released. It is meant to
// be called once, during construction: later calls, and a nil done, are
// no-ops.
func (e *Executable) WatchLifetime(done <-chan struct{}) {
	if done == nil {
		return
	}
	e.watchOnce.Do(func() {
		e.mu.Lock()
		e.lifetimeDone = done
		released := e.releasedLocked()
		e.mu.Unlock()
		go func() {
			select {
			case <-done:
				e.shutdown(context.Background())
			case <-released:
			}
		}()
	})
}

// WithPlugin calls fn with the compiled plugin and keeps the plugin open
// until fn returns: a concurrent Close does not release it before then.
// Concurrent WithPlugin calls do not block each other. It returns
// ErrExecutableClosed without calling fn once Close has been called, even
// while that Close is still waiting, or once the watched lifetime is done,
// ErrBytecodeNil if there is no plugin, and otherwise fn's error. If Close
// is waiting, or gave up waiting, when the last WithPlugin call returns,
// that call releases the plugin with ctx's values but not its cancellation,
// and counts as in flight until the release finishes. fn must not call
// Close on e with a ctx that is never done, since Close would wait for fn.
func (e *Executable) WithPlugin(ctx context.Context, fn func(plugin adapters.CompiledPlugin) error) error {
	e.mu.Lock()
	if e.closing || isClosed(e.lifetimeDone) {
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
// called releases the plugin and leaves inFlight only once that is done,
// so a waiting Close keeps honouring its ctx during a slow release.
func (e *Executable) done(ctx context.Context) {
	e.mu.Lock()
	last := e.closing && e.inFlight == 1
	if !last {
		e.inFlight--
	}
	e.mu.Unlock()
	if !last {
		return
	}
	e.release(ctx)
	e.mu.Lock()
	e.inFlight--
	e.mu.Unlock()
}

// readyToReleaseLocked reports whether a Close or the watcher can release
// the plugin now: Close has been called and nothing is in flight.
// releaseOnce makes sure only one caller releases it. e.mu must be held.
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

// release closes the plugin with ctx's values but not its cancellation, and
// then the released channel. It runs once; a caller that arrives while it
// runs waits for it to finish.
func (e *Executable) release(ctx context.Context) {
	e.releaseOnce.Do(func() {
		var err error
		if e.byteCode != nil {
			err = e.byteCode.Close(context.WithoutCancel(ctx))
		}
		e.mu.Lock()
		e.releaseErr = err
		ch := e.releasedLocked()
		e.mu.Unlock()
		close(ch)
	})
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

// Close releases the compiled plugin and its wazero runtime. Like
// [net/http.Server.Shutdown], it stops new work at once: from then on
// WithPlugin fails with ErrExecutableClosed. It then waits for in-flight
// WithPlugin calls (i.e. running Evals) until ctx is done. The last of them
// releases the plugin and counts as in flight until that release finishes:
// if it finishes first, Close returns the release error; if ctx is done
// first, Close returns an error wrapping ctx's error and the plugin is
// released when the last call returns. A Close that finds nothing in
// flight runs the release itself, or waits for one already under way, and
// returns its error regardless of ctx, so a plugin Close that hangs blocks
// it. The release never uses ctx's cancellation. Close is safe to call
// more than once and concurrently: the plugin is released once, and every
// call that sees the release finish returns its error. Call Close again
// after one that gave up waiting to learn whether the release failed. Safe
// on a nil Executable.
func (e *Executable) Close(ctx context.Context) error {
	if e == nil {
		return nil
	}
	released := e.shutdown(ctx)
	select {
	case <-released:
		return e.loadReleaseErr()
	case <-ctx.Done():
	}
	// ctx is done, but only in-flight calls, including the last one while it
	// releases, may make Close give up. A release that has already finished
	// wins. With nothing in flight, the release is under way without
	// cancellation, so wait for it.
	if e.hasInFlight() {
		if isClosed(released) {
			return e.loadReleaseErr()
		}
		return fmt.Errorf("extism: close stopped waiting for in-flight evals: %w", ctx.Err())
	}
	<-released
	return e.loadReleaseErr()
}

// shutdown marks e as closing and, with nothing in flight, releases the
// plugin before it returns; otherwise the last in-flight WithPlugin call
// releases it, and shutdown does not wait for that. It returns the channel
// closed once the plugin is released.
func (e *Executable) shutdown(ctx context.Context) <-chan struct{} {
	e.mu.Lock()
	released := e.releasedLocked()
	e.closing = true
	ready := e.readyToReleaseLocked()
	e.mu.Unlock()
	if ready {
		e.release(ctx)
	}
	return released
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
