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
	"testing"

	"github.com/tetratelabs/wazero"
)

// Run primes wazero's version cache by creating and closing one runtime,
// then runs the tests. It returns the exit code for os.Exit.
func Run(m *testing.M) int {
	ctx := context.Background()
	if err := wazero.NewRuntime(ctx).Close(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "wazero warm-up failed:", err)
		return 1
	}
	return m.Run()
}
