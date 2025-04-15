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
// For example, if the context key is "foo", the provider will store data
// at ctx.Value("foo") and retrieve it from the same location.
//
// Example:
//
//	provider := NewContextProvider(constants.EvalData)
//	ctx, _ := provider.AddDataToContext(ctx, map[string]any{"user": map[string]any{"name": "Alice"}})
//	// In script: name := ctx["user"]["name"]
func NewContextProvider(contextKey constants.ContextKey) *ContextProvider {
	return &ContextProvider{
		contextKey: contextKey,
	}
}

// GetData extracts a map[string]any data object from the context using the configured context key.
// It returns the entire context data as a map without any namespace filtering.
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

// AddDataToContext stores data in the context for script execution.
// It merges data from the provided maps, with later maps overriding earlier ones
// in case of key conflicts.
//
// The function recursively processes map entries to handle nested structures:
// - Maps are recursively merged at each level
// - HTTP Request objects are converted to maps using helpers.RequestToMap
// - Empty string keys are not allowed
//
// Example:
//
//	ctx := context.Background()
//	provider := NewContextProvider(constants.EvalData)
//
//	// Add multiple data maps
//	ctx, err := provider.AddDataToContext(ctx,
//	    map[string]any{"user": map[string]any{"name": "Alice"}},
//	    map[string]any{"request": httpRequest},
//	    map[string]any{"options": map[string]any{"debug": true}}
//	)
//
//	// In script, the data is accessed directly:
//	// userName := ctx["user"]["name"]          // "Alice"
//	// requestPath := ctx["request"]["URL_Path"]
//	// debugMode := ctx["options"]["debug"]     // true
func (p *ContextProvider) AddDataToContext(
	ctx context.Context,
	data ...map[string]any,
) (context.Context, error) {
	if p.contextKey == "" {
		return ctx, fmt.Errorf("context key is empty")
	}

	// Collect errors during processing
	var errz []error

	// Initialize the data storage map
	// First check if there's existing data in the context
	toStore := make(map[string]any)

	// Get existing data from context if available
	if existingData := ctx.Value(p.contextKey); existingData != nil {
		if existingMap, ok := existingData.(map[string]any); ok {
			// Deep copy the existing data to preserve it while allowing modifications
			maps.Copy(toStore, existingMap)
		}
	}

	// Process each input map
	for _, dataMap := range data {
		if dataMap == nil {
			continue
		}

		// Process each key-value pair in the input map
		for key, value := range dataMap {
			// Ensure key is not empty
			if key == "" {
				errz = append(errz, fmt.Errorf("empty keys are not allowed"))
				continue
			}

			// Process the value based on its type
			processedValue, err := p.processValue(value)
			if err != nil {
				errz = append(errz, fmt.Errorf("processing value for key '%s': %w", key, err))
				continue
			}

			// Handle merging the processed value with existing data
			p.mergeIntoMap(toStore, key, processedValue, &errz)
		}
	}

	// Always create a new context with whatever data we were able to process
	newCtx := context.WithValue(ctx, p.contextKey, toStore)

	// Return any errors that occurred (errors.Join returns nil if errz is empty)
	return newCtx, errors.Join(errz...)
}

// processValue handles different value types, converting them as needed
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
		// For maps, process each value recursively
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
		// For other types, just use the value as is
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
	// Handle maps specifically for recursive merging
	if newMap, ok := value.(map[string]any); ok {
		// If the key exists and is a map, merge recursively
		if existingValue, exists := target[key]; exists {
			if existingMap, ok := existingValue.(map[string]any); ok {
				// Both are maps, do a recursive merge
				for k, v := range newMap {
					p.mergeIntoMap(existingMap, k, v, errz)
				}
				return
			}
		}
	}

	// For non-map values or when the existing value is not a map
	// simply overwrite the value
	target[key] = value
}
