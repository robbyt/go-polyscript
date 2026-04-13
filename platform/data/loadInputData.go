package data

import (
	"context"
	"log/slog"
)

// LoadInputData retrieves input data using the given data provider.
// If the provider is nil, it returns an empty map. This function consolidates
// the common data-loading logic used across all engine evaluators.
func LoadInputData(
	ctx context.Context,
	logger *slog.Logger,
	provider Provider,
) (map[string]any, error) {
	// If no data provider, return empty map
	if provider == nil {
		logger.WarnContext(ctx, "no data provider available, using empty data")
		return make(map[string]any), nil
	}

	// Get input data from provider
	inputData, err := provider.GetData(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get input data from provider", "error", err)
		return nil, err
	}

	if len(inputData) == 0 {
		logger.WarnContext(ctx, "empty input data returned from provider")
	}
	logger.DebugContext(ctx, "input data loaded from provider", "inputData", inputData)
	return inputData, nil
}
