package data

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/platform/constants"
)

// ContextProvider retrieves and stores data in the context using a specified key.
type ContextProvider struct {
	contextKey constants.ContextKey
}

// NewContextProvider creates a new ContextProvider with the given context key.
// The context key determines where data is stored in the context object.
//
// See README.md for usage examples.
func NewContextProvider(contextKey constants.ContextKey) *ContextProvider {
	return &ContextProvider{
		contextKey: contextKey,
	}
}

// GetData extracts data from the context using the configured context key.
// It returns a deep copy (see deepCopyMap) of the stored map, so callers and
// script engines may mutate the result without affecting the context, its
// parents, or any context derived from it.
func (p *ContextProvider) GetData(ctx context.Context) (map[string]any, error) {
	if p.contextKey == "" {
		return nil, fmt.Errorf("context key is empty")
	}

	value := ctx.Value(p.contextKey)
	if value == nil {
		return make(map[string]any), nil
	}

	d, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input data type: expected map[string]any, got %T", value)
	}

	return deepCopyMap(d), nil
}

// AddDataToContext merges the provided maps into the context.
// Maps are recursively merged, HTTP Request objects are converted to maps,
// and later values override earlier ones for duplicate keys.
//
// See README.md for detailed usage examples.
//
// # Concurrency
//
// AddDataToContext is safe for concurrent use. Each call returns a
// derived context independent of its parent: data merged into the
// derived context does not affect the parent or any sibling derived
// chain. Existing data from the parent context is deep-copied at every
// nesting level when forming the derived context, so concurrent
// enrichments from goroutines that share a parent never race on shared
// inner map storage. GetData likewise returns a deep copy, so readers
// that mutate their result never touch the stored data.
//
// The deep-copy guarantee covers map[string]any and []any containers
// (see deepCopyValue). The incoming data is copied the same way:
// processValue rebuilds every map[string]any and []any it is given, so
// later changes to the caller's maps or slices do not reach the stored
// data. Other reference-typed values stored in the map
// (typed slices such as []string, pointers, channels, function values,
// and maps OTHER than map[string]any) remain shared with the source —
// ContextProvider never mutates them in place, but callers should avoid
// mutating them after handing them to AddDataToContext or receiving them
// from GetData. Value types (int, string, struct, etc.) are
// copied as usual when stored in an `any` slot. *http.Request and
// http.Request inputs are converted to map[string]any via
// helpers.RequestToMap on the way in, so the underlying request is not
// stored.
//
// The typical pattern is one derived-context chain per request, with
// the returned context threaded forward for any subsequent enrichment.
func (p *ContextProvider) AddDataToContext(
	ctx context.Context,
	data map[string]any,
) (context.Context, error) {
	if p.contextKey == "" {
		return ctx, fmt.Errorf("context key is empty")
	}

	var errz []error
	toStore := make(map[string]any)

	if existingData := ctx.Value(p.contextKey); existingData != nil {
		if existingMap, ok := existingData.(map[string]any); ok {
			// Deep-copy nested maps so subsequent in-place merge into
			// toStore cannot race with goroutines reading or merging the
			// same parent context. See the # Concurrency note above.
			toStore = deepCopyMap(existingMap)
		}
	}

	for key, value := range data {
		if key == "" {
			errz = append(errz, fmt.Errorf("empty keys are not allowed"))
			continue
		}

		processedValue, err := p.processValue(value)
		if err != nil {
			errz = append(errz, fmt.Errorf("processing value for key '%s': %w", key, err))
			continue
		}

		p.mergeIntoMap(toStore, key, processedValue, &errz)
	}

	newCtx := context.WithValue(ctx, p.contextKey, toStore)
	return newCtx, errors.Join(errz...)
}

// processValue converts values to appropriate types for storage. It
// recurses into map[string]any and []any, returning fresh containers, so
// nested maps are validated and requests are converted at any depth.
func (p *ContextProvider) processValue(value any) (any, error) {
	if value == nil {
		return nil, nil
	}

	switch v := value.(type) {
	case *http.Request:
		if v == nil {
			return nil, nil
		}
		return helpers.RequestToMap(v)
	case http.Request:
		return helpers.RequestToMap(&v)
	case map[string]any:
		// Handle maps by recursively processing their values
		result := make(map[string]any)
		for k, val := range v {
			if k == "" {
				return nil, fmt.Errorf("empty keys are not allowed in nested maps")
			}
			processedVal, err := p.processValue(val)
			if err != nil {
				return nil, fmt.Errorf("processing nested value for key '%s': %w", k, err)
			}
			result[k] = processedVal
		}
		return result, nil
	case []any:
		// Rebuild slices too, so the stored data shares no storage with the
		// caller's input and maps inside slices are validated and converted.
		if v == nil {
			return v, nil
		}
		result := make([]any, len(v))
		for i, elem := range v {
			processedElem, err := p.processValue(elem)
			if err != nil {
				return nil, fmt.Errorf("processing element %d: %w", i, err)
			}
			result[i] = processedElem
		}
		return result, nil
	default:
		return v, nil
	}
}

// mergeIntoMap recursively merges values into the target map
func (p *ContextProvider) mergeIntoMap(
	target map[string]any,
	key string,
	value any,
	errz *[]error,
) {
	// Special handling for map values - merge recursively
	if newMap, ok := value.(map[string]any); ok {
		if existingValue, exists := target[key]; exists {
			// A typed-nil existing map cannot be written to; the new map
			// replaces it below instead.
			if existingMap, ok := existingValue.(map[string]any); ok && existingMap != nil {
				for k, v := range newMap {
					p.mergeIntoMap(existingMap, k, v, errz)
				}
				return
			}
		}
	}

	// Non-map values simply replace existing values
	target[key] = value
}
