package snippets

// snippet:begin lifetime
import (
	"context"
	"time"

	"github.com/robbyt/go-polyscript"
)

// snippet:end

// lifetime is the README "Evaluator lifetime" fragment. It uses an engine
// that holds resources, so the Example fails if the evaluator is bound to
// loadCtx instead of appCtx.
func lifetime(appCtx context.Context, wasmBytes []byte) error {
	// snippet:begin lifetime
	// Give loading and compiling a deadline, but keep the evaluator
	// open until appCtx ends or Close is called.
	loadCtx, cancel := context.WithTimeout(appCtx, 10*time.Second)
	defer cancel()

	evaluator, err := polyscript.New[polyscript.Extism](
		loadCtx,
		polyscript.FromBytes(wasmBytes),
		polyscript.WithEntryPoint("greet"),
		polyscript.WithLifetime[polyscript.Extism](appCtx),
	)
	if err != nil {
		return err
	}
	defer evaluator.Close(appCtx)
	// snippet:end
	cancel()
	// Give a lifetime hook wrongly bound to loadCtx time to close the
	// evaluator before it is used.
	time.Sleep(20 * time.Millisecond)
	enrichedCtx, err := evaluator.AddDataToContext(appCtx, map[string]any{"input": "World"})
	return evalAndPrint(enrichedCtx, evaluator, err)
}
