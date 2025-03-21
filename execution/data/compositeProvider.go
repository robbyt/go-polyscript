package data

import (
	"context"
	"fmt"
	"maps"
)

// CompositeProvider combines multiple InputDataProviders and merges their results
// Later providers in the chain can override values from earlier providers
type CompositeProvider struct {
	// providers is the ordered list of providers to query
	providers []InputDataProvider
}

// NewCompositeProvider creates a new CompositeProvider with the given providers
// The providers will be queried in the order they are provided
func NewCompositeProvider(providers ...InputDataProvider) *CompositeProvider {
	return &CompositeProvider{
		providers: providers,
	}
}

// GetInputData implements InputDataProvider.GetInputData
// It calls each provider in sequence and merges the results
func (p *CompositeProvider) GetInputData(ctx context.Context) (map[string]any, error) {
	if len(p.providers) == 0 {
		return make(map[string]any), nil
	}

	// Start with an empty result
	result := make(map[string]any)

	// Process each provider and merge results
	for i, provider := range p.providers {
		if provider == nil {
			continue
		}

		// Get data from this provider
		data, err := provider.GetInputData(ctx)
		if err != nil {
			return nil, fmt.Errorf("error from provider %d: %w", i, err)
		}

		// Merge data into the result (overwrites existing keys)
		maps.Copy(result, data)
	}

	return result, nil
}
