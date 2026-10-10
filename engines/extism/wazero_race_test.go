package extism

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

// workaroundWazeroVersion is the wazero release that needs the TestMain
// warm-ups in the root and engines test packages, because of the version
// cache race fixed upstream in https://github.com/tetratelabs/wazero/pull/2536.
const workaroundWazeroVersion = "v1.12.0"

// TestWazeroRaceWorkaroundStillNeeded fails when the wazero dependency
// changes, as a reminder to reassess the race workaround.
func TestWazeroRaceWorkaroundStillNeeded(t *testing.T) {
	t.Parallel()

	info, ok := debug.ReadBuildInfo()
	require.True(t, ok, "build info should be available in test binaries")

	for _, dep := range info.Deps {
		if dep.Path != "github.com/tetratelabs/wazero" {
			continue
		}
		require.Nil(t, dep.Replace,
			"wazero is replaced (%v): reassess the race workaround for "+
				"tetratelabs/wazero#2536", dep.Replace)
		require.Equal(t, workaroundWazeroVersion, dep.Version,
			"wazero version changed: reassess the race workaround for "+
				"tetratelabs/wazero#2536. If this version includes the fix, delete "+
				"the TestMain warm-ups (main_test.go in the root and engines "+
				"packages), the known-issue notes in engines/README.md, README.md "+
				"and CHANGELOG.md, and this test.")
		return
	}
	t.Fatal("wazero not found in build info dependencies")
}
