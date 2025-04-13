package data

import (
	"context"
	"errors"
	"maps"
)

// ErrStaticProviderNoRuntimeUpdates is returned when trying to add runtime data to a StaticProvider.
// This is a sentinel error that can be checked by CompositeProvider to handle gracefully.
var ErrStaticProviderNoRuntimeUpdates = errors.New(
	"StaticProvider doesn't support adding data at runtime",
)

// StaticProvider supplies a predefined map of data.
// Useful for configuration values and testing.
type StaticProvider struct {
	data map[string]any
}

// NewStaticProvider creates a provider with fixed data.
// Initializes with an empty map if nil is provided.
func NewStaticProvider(data map[string]any) *StaticProvider {
	if len(data) == 0 {
		data = make(map[string]any)
	}
	return &StaticProvider{
		data: data,
	}
}

// GetData returns the static data map, cloned to prevent modification.
func (p *StaticProvider) GetData(_ context.Context) (map[string]any, error) {
	return maps.Clone(p.data), nil
}

// AddDataToContext returns a sentinel error as StaticProvider doesn't support dynamic data.
// Use a ContextProvider or CompositeProvider when runtime data updates are needed.
// The CompositeProvider should check for this specific error using errors.Is.
func (p *StaticProvider) AddDataToContext(
	ctx context.Context,
	data ...any,
) (context.Context, error) {
	return ctx, ErrStaticProviderNoRuntimeUpdates
}
