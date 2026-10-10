package wazerorace

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeM records whether Run was called and returns a fixed exit code.
type fakeM struct {
	ran  bool
	code int
}

func (f *fakeM) Run() int {
	f.ran = true
	return f.code
}

// TestRun swaps the package-level warmUp, so its subtests don't run in
// parallel.
func TestRun(t *testing.T) {
	t.Run("warms up and runs the tests", func(t *testing.T) {
		m := &fakeM{code: 3}
		assert.Equal(t, 3, Run(m), "Run should return the tests' exit code")
		assert.True(t, m.ran, "tests should run after the warm-up")
	})

	t.Run("failed warm-up skips the tests", func(t *testing.T) {
		orig := warmUp
		t.Cleanup(func() { warmUp = orig })
		warmUp = func() error { return errors.New("boom") }

		m := &fakeM{}
		assert.Equal(t, 1, Run(m), "Run should report failure")
		assert.False(t, m.ran, "tests should not run after a failed warm-up")
	})
}
