package extism

import (
	"os"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const wazeroModule = "github.com/tetratelabs/wazero"

// workaroundWazeroVersion is the wazero release that needs the TestMain
// warm-ups in the root and engines test packages, because of the version
// cache race fixed upstream in https://github.com/tetratelabs/wazero/pull/2536.
const workaroundWazeroVersion = "v1.12.0"

// TestWazeroRaceWorkaroundStillNeeded fails when the wazero dependency
// changes, as a reminder to reassess the race workaround.
func TestWazeroRaceWorkaroundStillNeeded(t *testing.T) {
	t.Parallel()

	version, replaced, ok := wazeroFromBuildInfo()
	if !ok {
		// Some Go versions don't record dependencies in test binaries.
		version, replaced = wazeroFromGoMod(t)
	}

	require.False(t, replaced,
		"wazero is replaced: reassess the race workaround for tetratelabs/wazero#2536")
	require.Equal(t, workaroundWazeroVersion, version,
		"wazero version changed: reassess the race workaround for "+
			"tetratelabs/wazero#2536. If this version includes the fix, delete "+
			"the TestMain warm-ups (main_test.go in the root and engines "+
			"packages), the known-issue notes in engines/README.md, README.md "+
			"and CHANGELOG.md, and this test.")
}

// wazeroFromBuildInfo returns the wazero version recorded in the test
// binary, whether it is replaced, and whether it was found.
func wazeroFromBuildInfo() (string, bool, bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false, false
	}
	for _, dep := range info.Deps {
		if dep.Path == wazeroModule {
			return dep.Version, dep.Replace != nil, true
		}
	}
	return "", false, false
}

// wazeroFromGoMod returns the wazero version required by the repository's
// go.mod and whether go.mod replaces it.
func wazeroFromGoMod(t *testing.T) (string, bool) {
	t.Helper()
	goMod, err := os.ReadFile("../../go.mod")
	require.NoError(t, err)

	var version string
	var replaced bool
	for line := range strings.Lines(string(goMod)) {
		fields := strings.Fields(line)
		if !strings.Contains(line, wazeroModule+" ") {
			continue
		}
		if strings.Contains(line, "=>") {
			replaced = true
			continue
		}
		for i, f := range fields {
			if f == wazeroModule && i+1 < len(fields) {
				version = fields[i+1]
			}
		}
	}
	require.NotEmpty(t, version, "wazero not found in go.mod")
	return version, replaced
}
