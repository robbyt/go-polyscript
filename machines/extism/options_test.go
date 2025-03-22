package extism

import (
	"testing"

	"github.com/robbyt/go-polyscript/machines/types"
	"github.com/robbyt/go-polyscript/options"
	"github.com/stretchr/testify/require"
)

func TestWithEntryPoint(t *testing.T) {
	// Test with correct machine type
	cfg1 := &options.Config{}
	opt := WithEntryPoint("main")
	err := opt(cfg1)
	require.NoError(t, err)

	// Check that options were set correctly
	compOpts, ok := cfg1.GetCompilerOptions().(*ExtismOptions)
	require.True(t, ok, "Expected *ExtismOptions")
	require.Equal(t, "main", compOpts.GetEntryPointName())

	// Test with explicit Extism machine type
	cfg2 := &options.Config{}
	cfg2.SetMachineType(types.Extism)
	err = opt(cfg2)
	require.NoError(t, err)

	// Test with wrong machine type
	cfg3 := &options.Config{}
	cfg3.SetMachineType(types.Starlark)
	err = opt(cfg3)
	require.Error(t, err)
	require.Contains(t, err.Error(), "can only be used with Extism machine")
}
