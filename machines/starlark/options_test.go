package starlark

import (
	"testing"

	"github.com/robbyt/go-polyscript/engine/options"
	"github.com/robbyt/go-polyscript/machines/types"
	"github.com/stretchr/testify/require"
)

func TestWithGlobals(t *testing.T) {
	// Test with correct machine type
	cfg1 := &options.Config{}
	opt := WithGlobals([]string{"ctx", "print"})
	err := opt(cfg1)
	require.NoError(t, err)

	// Check that options were set correctly
	compOpts, ok := cfg1.GetCompilerOptions().(*StarlarkOptions)
	require.True(t, ok, "Expected *StarlarkOptions")
	require.Equal(t, []string{"ctx", "print"}, compOpts.GetGlobals())

	// Test with explicit Starlark machine type
	cfg2 := &options.Config{}
	cfg2.SetMachineType(types.Starlark)
	err = opt(cfg2)
	require.NoError(t, err)

	// Test with wrong machine type
	cfg3 := &options.Config{}
	cfg3.SetMachineType(types.Risor)
	err = opt(cfg3)
	require.Error(t, err)
	require.Contains(t, err.Error(), "can only be used with Starlark machine")
}
