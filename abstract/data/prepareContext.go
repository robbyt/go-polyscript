package data

import (
	"context"
	"fmt"
	"log/slog"
)

// PrepareContextHelper is a utility function that implements the common logic for
// preparing a context with evaluation data. This function is used by various machine
// implementations to maintain consistent context preparation behavior.
//
// Parameters:
//   - ctx: The base context to enrich
//   - logger: A logger instance for recording operations
//   - provider: The data provider to use for storing data
//   - d: Variable list of data items to add to the context
//
// Returns:
//   - enrichedCtx: The context with added data
//   - err: Any error encountered during preparation
func PrepareContextHelper(
	ctx context.Context,
	logger *slog.Logger,
	provider Provider,
	d ...any,
) (context.Context, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if provider == nil {
		logger.WarnContext(ctx, "no data provider available for context preparation")
		return ctx, fmt.Errorf("no data provider available")
	}

	// Use the data provider plugin to store the raw data
	enrichedCtx, err := provider.AddDataToContext(ctx, d...)
	if err != nil {
		return ctx, fmt.Errorf("failed to prepare context: %w", err)
	}

	return enrichedCtx, err
}
