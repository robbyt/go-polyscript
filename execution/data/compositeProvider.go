package data

import (
	"context"
	"errors"
	"fmt"
	"maps"
)

// CompositeProvider combines multiple providers, with later providers
// overriding values from earlier ones in the chain.
type CompositeProvider struct {
	providers []Provider
}

// NewCompositeProvider creates a provider that queries given providers in order.
func NewCompositeProvider(providers ...Provider) *CompositeProvider {
	return &CompositeProvider{
		providers: providers,
	}
}

// GetData retrieves data from all providers and merges them into a single map.
// Queries providers in sequence, with later providers overriding values from earlier ones.
// Returns error on first provider failure.
func (p *CompositeProvider) GetData(ctx context.Context) (map[string]any, error) {
	result := make(map[string]any)

	for i, provider := range p.providers {
		if provider == nil {
			continue
		}

		data, err := provider.GetData(ctx)
		if err != nil {
			return nil, fmt.Errorf("error from provider %d: %w", i, err)
		}

		maps.Copy(result, data)
	}

	return result, nil
}

// AddDataToContext distributes data to all providers in the chain.
// Continues through all providers even if some fail, collecting errors with errors.Join.
// Prioritizes returning the most updated context possible.
//
// Example:
//
//	ctx := context.Background()
//	staticProvider := NewStaticProvider(map[string]any{"config": configData})
//	contextProvider := NewContextProvider(constants.EvalData)
//	composite := NewCompositeProvider(staticProvider, contextProvider)
//	ctx, err := composite.AddDataToContext(ctx, req, userData)
func (p *CompositeProvider) AddDataToContext(ctx context.Context, data ...any) (context.Context, error) {
	// Make a copy of the initial context
	currentCtx := ctx

	// Collect errors during processing
	var errz []error

	// Keep track of how many providers attempted to process
	providersAttempted := 0

	// Try to add data to each provider
	for i, provider := range p.providers {
		if provider == nil {
			continue
		}

		providersAttempted++

		// Pass data to this provider
		newCtx, err := provider.AddDataToContext(currentCtx, data...)

		if err != nil {
			// If a StaticProvider or other provider that doesn't support adding data
			// is in the chain, it will return an error. We'll collect these errors
			// but continue with other providers.
			errz = append(errz, fmt.Errorf("error from provider %d: %w", i, err))

			// Even with an error, the provider may have updated the context (partial success)
			// If the context is different, use it for the next provider
			if newCtx != currentCtx {
				currentCtx = newCtx
			}
			continue
		}

		// Update the context for the next provider
		currentCtx = newCtx
	}

	// If all providers failed AND we have providers, return the original context with errors
	if len(errz) == providersAttempted && providersAttempted > 0 {
		return ctx, errors.Join(errz...)
	}

	// Return the final context, along with any errors that occurred
	// errors.Join will return nil if errz is empty
	return currentCtx, errors.Join(errz...)
}
