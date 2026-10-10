// Package wazerorace works around a data race in wazero v1.12.0 for this
// module's tests. wazero looks up and caches its own version lazily on the
// first runtime creation, so tests that compile Extism modules in parallel
// race on that write under -race. Fixed upstream in
// https://github.com/tetratelabs/wazero/pull/2536 but not yet released.
//
// Every test package that creates wazero runtimes calls Run from its
// TestMain. engines/extism/wazero_race_test.go fails when wazero is
// upgraded, as a reminder to delete this package and those TestMains.
package wazerorace

import (
	"context"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"
)

// warmUp creates and closes one runtime so wazero caches its version. It is
// a variable so tests can make it fail.
var warmUp = func() error {
	ctx := context.Background()
	return wazero.NewRuntime(ctx).Close(ctx)
}

// Run primes wazero's version cache, then runs the tests by calling m.Run.
// Pass the *testing.M from TestMain. It returns the exit code for os.Exit.
func Run(m interface{ Run() int }) int {
	if err := warmUp(); err != nil {
		fmt.Fprintln(os.Stderr, "wazero warm-up failed:", err)
		return 1
	}
	return m.Run()
}
