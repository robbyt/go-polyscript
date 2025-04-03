package data

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/internal/helpers"
)

// ContextProvider retrieves and stores data in the context using a specified key.
type ContextProvider struct {
	contextKey  constants.ContextKey
	storageKey  string
	requestKey  string
	responseKey string
}

// NewContextProvider creates a new ContextProvider with the given context key.
// For example, if the context key is "foo", the provider will store the input data under the
// ctx.Value("foo") key. Later, lookup will also use the same key to retrieve the data from the
// context object.
func NewContextProvider(contextKey constants.ContextKey) *ContextProvider {
	return &ContextProvider{
		contextKey: contextKey,

		// hard-coded storage keys for accessing the data in the final ctx object.
		storageKey:  constants.InputData, // ctx["input_data"]
		requestKey:  constants.Request,   // ctx["request"]
		responseKey: constants.Response,  // ctx["response"]
	}
}

// GetData extracts a map[string]any data object from the context using the previously configured context key.
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
// Prioritizes consistent data structure for scripts over error propagation,
// ensuring scripts always have required data structures available.
//
// Example:
//
//	ctx := context.Background()
//	provider := NewContextProvider(constants.EvalData)
//	req := &http.Request{...}
//	scriptData := map[string]any{"user": "admin"}
//	ctx, err := provider.AddDataToContext(ctx, req, scriptData)
func (p *ContextProvider) AddDataToContext(
	ctx context.Context,
	data ...any,
) (context.Context, error) {
	if p.contextKey == "" {
		return ctx, fmt.Errorf("context key is empty")
	}

	// Collect errors during processing
	var errz []error

	// Initialize the data storage map
	toStore := make(map[string]any)

	// Process each data item based on its type
	for _, item := range data {
		if item == nil {
			continue
		}

		switch v := item.(type) {
		default:
			// For unhandled types, log an error and continue
			errz = append(errz, fmt.Errorf("unsupported data type for ContextProvider: %T", item))
			continue
		case map[string]any:
			scriptData := make(map[string]any)

			// Reuse existing data map if available, because we're iterating multiple data
			if existingScriptData, ok := toStore[p.storageKey].(map[string]any); ok {
				scriptData = existingScriptData
			}

			// Copy new data into the map (overwriting any existing keys)
			maps.Copy(scriptData, v)
			toStore[p.storageKey] = scriptData
		case *http.Request:
			if v == nil {
				continue
			}

			if existingValue, exists := toStore[p.requestKey]; exists {
				errz = append(errz, fmt.Errorf("request data already set: %v", existingValue))
				continue
			}

			reqMap, err := helpers.RequestToMap(v)
			if err != nil {
				errz = append(errz, fmt.Errorf("failed to convert HTTP request to map: %w", err))
				continue
			}
			toStore[p.requestKey] = reqMap
		case http.Request:
			if existingValue, exists := toStore[p.requestKey]; exists {
				errz = append(errz, fmt.Errorf("request data already set: %v", existingValue))
				continue
			}

			reqMap, err := helpers.RequestToMap(&v)
			if err != nil {
				errz = append(errz, fmt.Errorf("failed to convert HTTP request to map: %w", err))
				continue
			}
			toStore[p.requestKey] = reqMap
			/*
				TODO: add helpers.ResponseToMap
				case *http.Response:
					if v == nil {
						continue
					}

					if existingValue, exists := toStore[p.responseKey]; exists {
						errz = append(errz, fmt.Errorf("response data already set: %v", existingValue))
						continue
					}

					respMap, err := helpers.ResponseToMap(v)
					if err != nil {
						errz = append(errz, fmt.Errorf("failed to convert HTTP response to map: %w", err))
						continue
					}
					toStore[p.responseKey] = respMap
				case http.Response:
					if existingValue, exists := toStore[p.responseKey]; exists {
						errz = append(errz, fmt.Errorf("response data already set: %v", existingValue))
						continue
					}

					respMap, err := helpers.ResponseToMap(&v)
					if err != nil {
						errz = append(errz, fmt.Errorf("failed to convert HTTP response to map: %w", err))
						continue
					}
					toStore[p.responseKey] = respMap
			*/
		}
	}

	// Always create a new context with whatever data we were able to process
	newCtx := context.WithValue(ctx, p.contextKey, toStore)

	// Return any errors that occurred (errors.Join returns nil if errz is empty)
	// Even with errors, we return the updated context
	return newCtx, errors.Join(errz...)
}
