package data

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/robbyt/go-polyscript/internal/helpers"
)

// AddDataToContextHelper is a utility function that implements the common logic for
// adding data to a context for evaluation. This function is used by various engine
// implementations to maintain consistent data handling behavior.
//
// Pass nil for handler to inherit from slog.Default() via [helpers.SetupLogger];
// pass an explicit slog.Handler to honor the host's configuration. Records emitted
// from this function carry an "AddDataToContext" sub-group.
//
// Parameters:
//   - ctx: The base context to enrich
//   - handler: A slog.Handler for diagnostic logging (nil = inherit from slog.Default())
//   - provider: The data provider to use for storing data
//   - d: The data map to add to the context
//
// Returns:
//   - enrichedCtx: The context with added data
//   - err: Any error encountered during the operation
func AddDataToContextHelper(
	ctx context.Context,
	handler slog.Handler,
	provider Provider,
	d map[string]any,
) (context.Context, error) {
	_, logger := helpers.SetupLogger(handler, "data", "AddDataToContext")

	if provider == nil {
		logger.WarnContext(ctx, "no data provider available for context preparation")
		return ctx, fmt.Errorf("no data provider available")
	}

	// Use the data provider plugin to store the raw data
	enrichedCtx, err := provider.AddDataToContext(ctx, d)
	if err != nil {
		return ctx, fmt.Errorf("failed to prepare context: %w", err)
	}

	return enrichedCtx, err
}

// AddDataToContextFromProvider is a convenience wrapper that handles a nil provider
// by returning a clear error, then delegates to AddDataToContextHelper.
// This consolidates the nil-check + delegation pattern duplicated across all engine evaluators.
//
// Pass nil for handler to inherit from slog.Default() via [helpers.SetupLogger];
// pass an explicit slog.Handler to honor the host's configuration.
func AddDataToContextFromProvider(
	ctx context.Context,
	handler slog.Handler,
	provider Provider,
	d map[string]any,
) (context.Context, error) {
	if provider == nil {
		return ctx, fmt.Errorf("no data provider available")
	}
	return AddDataToContextHelper(ctx, handler, provider, d)
}
