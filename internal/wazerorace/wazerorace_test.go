package wazerorace

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// stubWarmUp replaces warmUp and resets the Prime state for one test,
// restoring both afterwards.
func stubWarmUp(t *testing.T, fn func() error) {
	t.Helper()
	orig := warmUp
	reset := func() {
		primeOnce = sync.Once{}
		primeErr = nil
	}
	t.Cleanup(func() {
		warmUp = orig
		reset()
	})
	reset()
	warmUp = fn
}

// The tests below swap package-level state, so they don't run in parallel.

func TestPrime(t *testing.T) {
	t.Run("warms up once across concurrent callers", func(t *testing.T) {
		var calls atomic.Int32
		stubWarmUp(t, func() error {
			calls.Add(1)
			return nil
		})

		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() { assert.NoError(t, Prime()) })
		}
		wg.Wait()
		assert.Equal(t, int32(1), calls.Load(), "warm-up should run exactly once")
	})

	t.Run("reports a failed warm-up to every caller", func(t *testing.T) {
		errBoom := errors.New("boom")
		stubWarmUp(t, func() error { return errBoom })

		require.ErrorIs(t, Prime(), errBoom)
		require.ErrorIs(t, Prime(), errBoom, "later callers should see the same error")
	})

	t.Run("real warm-up succeeds", func(t *testing.T) {
		stubWarmUp(t, warmUp)
		require.NoError(t, Prime())
	})
}

func TestRun(t *testing.T) {
	t.Run("warms up and runs the tests", func(t *testing.T) {
		stubWarmUp(t, func() error { return nil })
		m := &fakeM{code: 3}
		assert.Equal(t, 3, Run(m), "Run should return the tests' exit code")
		assert.True(t, m.ran, "tests should run after the warm-up")
	})

	t.Run("failed warm-up skips the tests", func(t *testing.T) {
		stubWarmUp(t, func() error { return errors.New("boom") })
		m := &fakeM{}
		assert.Equal(t, 1, Run(m), "Run should report failure")
		assert.False(t, m.ran, "tests should not run after a failed warm-up")
	})
}
