package data

import (
	"context"
	"errors"
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
// It stores a deep copy (see deepCopyMap) of data, so later changes to the
// caller's map do not affect the provider. A nil map yields an empty provider.
func NewStaticProvider(data map[string]any) *StaticProvider {
	return &StaticProvider{
		data: deepCopyMap(data),
	}
}

// GetData returns a deep copy (see deepCopyMap) of the static data, so
// callers and script engines may mutate the result, including nested maps
// and []any slices, without affecting the provider.
func (p *StaticProvider) GetData(_ context.Context) (map[string]any, error) {
	return deepCopyMap(p.data), nil
}

// AddDataToContext returns a sentinel error as StaticProvider doesn't support dynamic data.
// Use a ContextProvider or CompositeProvider when runtime data updates are needed.
// The CompositeProvider should check for this specific error using errors.Is.
func (p *StaticProvider) AddDataToContext(
	ctx context.Context,
	_ map[string]any,
) (context.Context, error) {
	return ctx, ErrStaticProviderNoRuntimeUpdates
}
