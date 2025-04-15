package data

import (
	"context"
	"errors"
	"fmt"
	"maps"
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

	return d, nil
}

// AddDataToContext merges the provided maps into the context.
// Maps are recursively merged, HTTP Request objects are converted to maps,
// and later values override earlier ones for duplicate keys.
//
// See README.md for detailed usage examples.
func (p *ContextProvider) AddDataToContext(
	ctx context.Context,
	data ...map[string]any,
) (context.Context, error) {
	if p.contextKey == "" {
		return ctx, fmt.Errorf("context key is empty")
	}

	var errz []error
	toStore := make(map[string]any)

	if existingData := ctx.Value(p.contextKey); existingData != nil {
		if existingMap, ok := existingData.(map[string]any); ok {
			maps.Copy(toStore, existingMap)
		}
	}

	for _, dataMap := range data {
		if dataMap == nil {
			continue
		}

		for key, value := range dataMap {
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
	}

	newCtx := context.WithValue(ctx, p.contextKey, toStore)
	return newCtx, errors.Join(errz...)
}

// processValue converts values to appropriate types for storage
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
			if existingMap, ok := existingValue.(map[string]any); ok {
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
