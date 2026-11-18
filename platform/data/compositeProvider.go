package data

import (
	"context"
	"errors"
	"fmt"
	"maps"
)

// ErrNoProviders is returned by CompositeProvider.AddDataToContext when the
// composite holds no non-nil providers, so the data has nowhere to be stored.
var ErrNoProviders = errors.New("CompositeProvider has no providers to store data")

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
// Performs deep merging of nested maps for proper data composition.
// Returns error on first provider failure.
//
// The merged map shares no mutable storage with the providers as long as
// each provider's GetData returns a map the caller owns, as StaticProvider
// and ContextProvider do. deepMerge stores values found in only one
// provider by reference, so a custom provider that hands out its internal
// map would be aliased by the result.
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

		// Use deepMerge for proper handling of nested structures
		result = deepMerge(result, data)
	}

	return result, nil
}

// deepMerge recursively merges map[string]any maps. Values from dst override those from src.
// Special handling for nested maps to do a deep merge rather than simple replacement.
// Arrays and other data types are replaced entirely, not merged.
func deepMerge(src, dst map[string]any) map[string]any {
	result := maps.Clone(src)
	// maps.Clone(nil) is nil, which a typed-nil nested map in src reaches
	// via the recursion below; writing to it would panic.
	if result == nil {
		result = make(map[string]any, len(dst))
	}

	for k, dstVal := range dst {
		srcVal, exists := result[k]

		// If the key doesn't exist in source, just use destination value
		if !exists {
			result[k] = dstVal
			continue
		}

		// If both values are maps, merge them recursively
		srcMap, srcIsMap := srcVal.(map[string]any)
		dstMap, dstIsMap := dstVal.(map[string]any)

		if srcIsMap && dstIsMap {
			// Recursively merge nested maps
			result[k] = deepMerge(srcMap, dstMap)
		} else {
			// For non-map types, destination value overrides source
			result[k] = dstVal
		}
	}

	return result
}

// AddDataToContext distributes data to all providers in the chain, passing
// each successful provider's returned context on to the next one. It
// continues through all providers even if some fail; a failing provider's
// returned context is discarded.
//
// Providers are judged by the error they return, not by their type, so a
// nested CompositeProvider or a wrapper around a StaticProvider is treated
// like the providers it holds. Two errors are not failures:
//   - ErrStaticProviderNoRuntimeUpdates (from a StaticProvider, or from a
//     nested CompositeProvider holding only StaticProviders) is an expected
//     rejection: that provider never stores runtime data.
//   - ErrNoProviders (from a nested CompositeProvider with no providers)
//     means there was nothing to store into, the same as a nil slot.
//
// An error counts as one of these only when every error it wraps (through
// fmt.Errorf %w or errors.Join) is that sentinel; an error that also wraps
// a real failure is a real failure.
//
// The result is decided once every provider has been tried:
//   - Any provider failed: the original ctx and all the failures joined
//     with errors.Join. Expected rejections are not included. The contexts
//     built by the providers that succeeded are discarded, so a nested
//     composite never hands partial data to its parent.
//   - Otherwise, if at least one provider stored the data: the most
//     updated context and nil.
//   - Otherwise, if some provider rejected the data as static: the
//     original ctx and the joined ErrStaticProviderNoRuntimeUpdates errors.
//   - Otherwise there was nothing to store into (no providers, only nil
//     slots, or only empty nested composites): the original ctx and
//     ErrNoProviders.
//
// Returning the original ctx is not a rollback. Every provider is still
// called after a failure, and anything a provider did besides building a
// context is not undone: a ContextProvider may already have read an HTTP
// request body, for example. Only the derived contexts are thrown away.
//
// Example:
//
//	ctx := context.Background()
//	staticProvider := data.NewStaticProvider(map[string]any{"config": configData})
//	contextProvider := data.NewContextProvider(constants.EvalData)
//	composite := data.NewCompositeProvider(staticProvider, contextProvider)
//	ctx, err := composite.AddDataToContext(ctx, userData)
func (p *CompositeProvider) AddDataToContext(
	ctx context.Context,
	data map[string]any,
) (context.Context, error) {
	// Start with the original context
	finalCtx := ctx
	var tally addDataTally

	// Try to add data to each provider
	for i, provider := range p.providers {
		if provider == nil {
			continue
		}

		nextCtx, err := provider.AddDataToContext(finalCtx, data)
		if err != nil {
			// A failing provider's returned context is discarded.
			tally.recordError(i, err)
			continue
		}

		tally.stored++
		finalCtx = nextCtx
	}

	if err := tally.finalErr(); err != nil {
		// Discard every derived context, so callers never see partial data.
		return ctx, err
	}
	return finalCtx, nil
}

// addDataTally accumulates the per-provider outcomes of
// CompositeProvider.AddDataToContext so the final result can be decided
// after every provider has been tried.
type addDataTally struct {
	stored     int     // providers that accepted the data
	errs       []error // real failures
	staticErrs []error // expected ErrStaticProviderNoRuntimeUpdates rejections
}

// recordError wraps err with the provider's index in the providers list
// (nil slots included) and files it as an expected static rejection, a
// real failure, or, for a provider with nothing to store into, not at all.
func (t *addDataTally) recordError(i int, err error) {
	switch {
	case onlyWraps(err, ErrNoProviders):
		// e.g. an empty nested CompositeProvider; treated like a nil slot.
	case onlyWraps(err, ErrStaticProviderNoRuntimeUpdates):
		t.staticErrs = append(t.staticErrs, fmt.Errorf("error from provider %d: %w", i, err))
	default:
		t.errs = append(t.errs, fmt.Errorf("error from provider %d: %w", i, err))
	}
}

// finalErr returns the error AddDataToContext should report alongside the
// original ctx, or nil when every provider that can store data did.
func (t *addDataTally) finalErr() error {
	switch {
	case len(t.errs) > 0:
		// Any real failure is reported, even if other providers succeeded.
		return errors.Join(t.errs...)
	case t.stored > 0:
		// Static rejections are expected when another provider stored the data.
		return nil
	case len(t.staticErrs) > 0:
		// Every provider that could have taken the data is static.
		return errors.Join(t.staticErrs...)
	default:
		// Nothing to store the data in; don't let it vanish silently.
		return ErrNoProviders
	}
}

// onlyWraps reports whether every leaf of err's wrap tree, followed through
// Unwrap() error and Unwrap() []error, is target. Unlike errors.Is, it is
// false for an error that wraps target alongside an unrelated failure.
func onlyWraps(err, target error) bool {
	switch e := err.(type) { //nolint:errorlint // walks the wrap tree by hand on purpose
	case nil:
		return false
	case interface{ Unwrap() []error }:
		errs := e.Unwrap()
		if len(errs) == 0 {
			return false
		}
		for _, inner := range errs {
			if !onlyWraps(inner, target) {
				return false
			}
		}
		return true
	case interface{ Unwrap() error }:
		return onlyWraps(e.Unwrap(), target)
	default:
		return errors.Is(err, target)
	}
}
