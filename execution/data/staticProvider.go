package data

import (
	"context"
	"fmt"
	"maps"
)

// StaticProvider supplies a predefined map of data.
// Useful for configuration values and testing.
type StaticProvider struct {
	data map[string]any
}

// NewStaticProvider creates a provider with fixed data.
// Initializes with an empty map if nil is provided.
func NewStaticProvider(data map[string]any) *StaticProvider {
	if data == nil {
		data = make(map[string]any)
	}
	return &StaticProvider{
		data: data,
	}
}

// GetData returns the static data map, cloned to prevent modification.
func (p *StaticProvider) GetData(ctx context.Context) (map[string]any, error) {
	return maps.Clone(p.data), nil
}

// AddDataToContext returns an error as StaticProvider doesn't support dynamic data.
// Use a ContextProvider or CompositeProvider when runtime data updates are needed.
func (p *StaticProvider) AddDataToContext(ctx context.Context, data ...any) (context.Context, error) {
	return ctx, fmt.Errorf("StaticProvider doesn't support adding data at runtime")
}
