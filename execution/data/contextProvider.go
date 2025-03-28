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
	contextKey constants.ContextKey
}

// NewContextProvider creates a new ContextProvider with the given context key.
func NewContextProvider(contextKey constants.ContextKey) *ContextProvider {
	return &ContextProvider{
		contextKey: contextKey,
	}
}

// GetData extracts a map[string]any from the context using the configured key.
func (p *ContextProvider) GetData(ctx context.Context) (map[string]any, error) {
	if p.contextKey == "" {
		return nil, fmt.Errorf("context key is empty")
	}

	value := ctx.Value(p.contextKey)
	if value == nil {
		return make(map[string]any), nil
	}

	inputData, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input data type: expected map[string]any, got %T", value)
	}

	return inputData, nil
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

	// Get existing data from context if any
	existingData, err := p.GetData(ctx)
	if err == nil && existingData != nil && len(existingData) > 0 {
		maps.Copy(toStore, existingData)
	}

	// Initialize standard keys with empty maps if they don't exist
	// This ensures scripts always have a consistent data structure to work with
	if _, exists := toStore[constants.Request]; !exists {
		toStore[constants.Request] = make(map[string]any)
	}
	if _, exists := toStore[constants.ScriptData]; !exists {
		toStore[constants.ScriptData] = make(map[string]any)
	}

	// Process each data item based on its type
	for _, item := range data {
		if item == nil {
			continue
		}

		switch v := item.(type) {
		case *http.Request:
			// Handle HTTP request - convert to map and store under request key
			reqMap, err := helpers.RequestToMap(v)
			if err != nil {
				errz = append(errz, fmt.Errorf("failed to convert HTTP request to map: %w", err))
				// Keep the empty request map - don't skip storing data
				// The empty map was already initialized above
			} else {
				toStore[constants.Request] = reqMap
			}

		case http.Request:
			// Handle HTTP request value (not pointer)
			reqMap, err := helpers.RequestToMap(&v)
			if err != nil {
				errz = append(errz, fmt.Errorf("failed to convert HTTP request to map: %w", err))
				// Keep the empty request map - don't skip storing data
				// The empty map was already initialized above
			} else {
				toStore[constants.Request] = reqMap
			}

		case map[string]any:
			// Handle script data map - store under script_data key
			// We merge with existing script data rather than replacing it
			scriptData := make(map[string]any)
			if existingScriptData, ok := toStore[constants.ScriptData].(map[string]any); ok {
				maps.Copy(scriptData, existingScriptData)
			}
			maps.Copy(scriptData, v)
			toStore[constants.ScriptData] = scriptData

		default:
			// For unrecognized types, log warning or return error
			errz = append(errz, fmt.Errorf("unsupported data type for ContextProvider: %T", item))
			// We continue processing other items even if this one failed
		}
	}

	// Always create a new context with whatever data we were able to process
	newCtx := context.WithValue(ctx, p.contextKey, toStore)

	// Return any errors that occurred (errors.Join returns nil if errz is empty)
	// Even with errors, we return the updated context
	return newCtx, errors.Join(errz...)
}
