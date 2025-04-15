package data

import (
	"context"
	"fmt"
	"log/slog"
)

// AddDataToContextHelper is a utility function that implements the common logic for
// adding data to a context for evaluation. This function is used by various engine
// implementations to maintain consistent data handling behavior.
//
// Parameters:
//   - ctx: The base context to enrich
//   - logger: A logger instance for recording operations
//   - provider: The data provider to use for storing data
//   - d: Variable list of data items to add to the context
//
// Returns:
//   - enrichedCtx: The context with added data
//   - err: Any error encountered during the operation
func AddDataToContextHelper(
	ctx context.Context,
	logger *slog.Logger,
	provider Provider,
	d ...map[string]any,
) (context.Context, error) {
	if logger == nil {
		// TODO: remove or use logger more effectively
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
