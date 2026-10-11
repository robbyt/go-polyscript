package snippets

// snippet:begin logging
import (
	"context"
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript"
)

// snippet:end

// captureLogs is the README "Capturing diagnostic logs" fragment.
func captureLogs(script string, data map[string]any) error {
	// snippet:begin logging
	ctx := context.Background()
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})

	evaluator, err := polyscript.New[polyscript.Risor](
		ctx,
		polyscript.FromString(script),
		polyscript.WithLogHandler[polyscript.Risor](handler),
		polyscript.WithStaticData[polyscript.Risor](data),
	)
	// snippet:end
	return evalAndPrint(ctx, evaluator, err)
}
