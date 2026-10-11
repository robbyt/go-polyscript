// Package wazerorace works around a data race in wazero v1.12.0. wazero
// looks up and caches its own version lazily on the first runtime creation,
// so two runtimes created at the same moment race on that write under
// -race. Fixed upstream in https://github.com/tetratelabs/wazero/pull/2536
// but not yet released.
//
// The Extism compiler calls Prime before creating a runtime, so concurrent
// compiles in an app or a test are safe. Tests that create wazero runtimes
// without going through the compiler call Run from their TestMain.
// engines/extism/wazero_race_test.go fails when wazero is upgraded, as a
// reminder to delete this package and its callers.
package wazerorace

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/tetratelabs/wazero"
)

// warmUp creates and closes one runtime so wazero caches its version. It is
// a variable so tests can make it fail.
var warmUp = func() error {
	ctx := context.Background()
	return wazero.NewRuntime(ctx).Close(ctx)
}

var (
	primeOnce sync.Once
	primeErr  error
)

// Prime caches wazero's version once per process. It is safe to call
// concurrently: callers after the first wait until the version is cached,
// so their runtime creations never race with that write.
func Prime() error {
	primeOnce.Do(func() { primeErr = warmUp() })
	return primeErr
}

// Run primes wazero's version cache, then runs the tests by calling m.Run.
// Pass the *testing.M from TestMain. It returns the exit code for os.Exit.
func Run(m interface{ Run() int }) int {
	if err := Prime(); err != nil {
		fmt.Fprintln(os.Stderr, "wazero warm-up failed:", err)
		return 1
	}
	return m.Run()
}
