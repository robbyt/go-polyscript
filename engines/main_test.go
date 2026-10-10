package engines

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/tetratelabs/wazero"
)

// TestMain primes wazero's version cache before any test runs. wazero
// v1.12.0 looks up and caches its own version lazily on the first runtime
// creation, so parallel tests that compile Extism modules race on that write
// under -race. Fixed upstream in https://github.com/tetratelabs/wazero/pull/2536
// but not yet released; engines/extism/wazero_race_test.go fails when wazero
// is upgraded, as a reminder to remove this workaround.
func TestMain(m *testing.M) {
	ctx := context.Background()
	if err := wazero.NewRuntime(ctx).Close(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "wazero warm-up failed:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
