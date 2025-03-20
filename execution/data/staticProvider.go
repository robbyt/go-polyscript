package data

import (
	"context"
	"maps"
)

// StaticProvider is a simple provider that returns a predefined map of data
// It's useful for testing and for cases where the input data is known in advance
// and doesn't need to be retrieved from the context or external sources.
type StaticProvider struct {
	// data is the static map of data that will be returned by GetInputData
	data map[string]any
}

// NewStaticProvider creates a new StaticProvider with the provided data map
func NewStaticProvider(data map[string]any) *StaticProvider {
	if data == nil {
		data = make(map[string]any)
	}
	return &StaticProvider{
		data: data,
	}
}

// GetInputData implements InputDataProvider.GetInputData
// It simply returns the static data map regardless of the context
func (p *StaticProvider) GetInputData(ctx context.Context) (map[string]any, error) {
	// Return a clone of the data to prevent modification of the original
	return maps.Clone(p.data), nil
}
