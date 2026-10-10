package extism

import (
	"context"
	"testing"
	"time"

	"github.com/robbyt/go-polyscript/engines/extism/compiler"
	"github.com/robbyt/go-polyscript/engines/extism/evaluator"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script"
	"github.com/robbyt/go-polyscript/platform/script/loader"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

// spinWASM is a minimal hand-encoded WASM module exporting "spin", a
// function that never returns. Equivalent WAT:
//
//	(module (func (export "spin") (result i32) (loop (br 0)) (i32.const 0)))
var spinWASM = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, // magic, version
	0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7f, // type: () -> i32
	0x03, 0x02, 0x01, 0x00, // function 0 has type 0
	0x07, 0x08, 0x01, 0x04, 's', 'p', 'i', 'n', 0x00, 0x00, // export "spin"
	0x0a, 0x0b, 0x01, 0x09, 0x00, // code: one body, no locals
	0x03, 0x40, 0x0c, 0x00, 0x0b, // loop br 0 end
	0x41, 0x00, 0x0b, // i32.const 0 end
}

// TestEval_CancelStopsRunningGuest verifies that an expired Eval ctx stops a
// guest that would otherwise run forever, including when the caller supplies
// their own wazero runtime config.
func TestEval_CancelStopsRunningGuest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []compiler.FunctionalOption
	}{
		{name: "default runtime config"},
		{
			name: "custom runtime config",
			opts: []compiler.FunctionalOption{
				compiler.WithRuntimeConfig(wazero.NewRuntimeConfig()),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ldr, err := loader.NewFromBytes(spinWASM)
			require.NoError(t, err)

			comp, err := NewCompiler(
				append([]compiler.FunctionalOption{compiler.WithEntryPoint("spin")}, tc.opts...)...,
			)
			require.NoError(t, err)

			execUnit, err := script.NewExecutableUnit(
				t.Context(), nil, "spin", ldr, comp,
				data.NewContextProvider(constants.EvalData),
			)
			require.NoError(t, err)
			eval := evaluator.New(nil, execUnit)

			ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
			defer cancel()

			// Without termination checks compiled in, the guest spins in
			// native code that Go cannot preempt, so a regression hangs the
			// whole test binary until go test's -timeout rather than
			// failing here.
			_, err = eval.Eval(ctx)
			require.ErrorIs(t, err, context.DeadlineExceeded)
		})
	}
}
