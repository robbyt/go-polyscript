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
// Performs deep merging of nested maps for proper data composition.
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

// AddDataToContext distributes data to all providers in the chain.
// Continues through all providers even if some fail.
// StaticProvider errors are handled specially based on context.
//
// Example:
//
//	ctx := context.Background()
//	staticProvider := NewStaticProvider(map[string]any{"config": configData})
//	contextProvider := NewContextProvider(constants.EvalData)
//	composite := NewCompositeProvider(staticProvider, contextProvider)
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

		// Check if this is a StaticProvider (which always returns errors on AddDataToContext)
		_, isStaticProvider := provider.(*StaticProvider)
		tally.countProvider(isStaticProvider)

		nextCtx, err := provider.AddDataToContext(finalCtx, data)
		if err != nil {
			// A failing provider's returned context is discarded.
			tally.recordError(i, isStaticProvider, err)
			continue
		}

		// Success - update the context and count
		finalCtx = nextCtx
		tally.recordSuccess()
	}

	if err := tally.finalErr(); err != nil {
		return ctx, err
	}

	// Return the most updated context with no error
	return finalCtx, nil
}

// addDataTally accumulates the per-provider outcomes of
// CompositeProvider.AddDataToContext so the final result can be decided
// after every provider has been tried.
type addDataTally struct {
	errs         []error // errors from non-static providers (and any non-sentinel StaticProvider error)
	staticErrs   []error // ErrStaticProviderNoRuntimeUpdates errors from StaticProviders
	successCount int     // providers that returned no error
	totalCount   int     // non-nil providers that are not a *StaticProvider
	staticCount  int     // providers that are a *StaticProvider
}

// countProvider records that a non-nil provider is about to be tried.
// StaticProviders are counted separately because they always reject runtime data.
func (t *addDataTally) countProvider(isStaticProvider bool) {
	if isStaticProvider {
		t.staticCount++
		return
	}
	t.totalCount++
}

// recordSuccess records that a provider accepted the data.
func (t *addDataTally) recordSuccess() {
	t.successCount++
}

// recordError wraps err with the provider's index in the providers list
// (nil slots included) and files it as either an expected StaticProvider
// rejection or a regular provider error.
func (t *addDataTally) recordError(i int, isStaticProvider bool, err error) {
	wrapped := fmt.Errorf("error from provider %d: %w", i, err)
	if isStaticProvider && errors.Is(err, ErrStaticProviderNoRuntimeUpdates) {
		t.staticErrs = append(t.staticErrs, wrapped)
		return
	}
	t.errs = append(t.errs, wrapped)
}

// finalErr returns the error AddDataToContext should report, or nil when
// the accumulated context should be returned.
func (t *addDataTally) finalErr() error {
	// If every non-nil provider is a StaticProvider, they all rejected the
	// runtime data, so report their errors
	if t.staticCount > 0 && t.totalCount == 0 && len(t.staticErrs) > 0 {
		return errors.Join(t.staticErrs...)
	}

	// If all non-StaticProvider providers failed, return an error
	if t.totalCount > 0 && t.successCount == 0 && len(t.errs) > 0 {
		return errors.Join(t.errs...)
	}

	return nil
}
