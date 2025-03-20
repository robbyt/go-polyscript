package data

import (
	"context"
	"fmt"

	"github.com/robbyt/go-polyscript/execution/constants"
)

// Using constants.ContextKey for context keys to maintain type safety

// ContextProvider retrieves input data from the context using a specified key
// This is the default provider and maintains backward compatibility with
// the existing implementation that uses context.Value to store input data.
type ContextProvider struct {
	// contextKey is the key used to retrieve data from the context
	contextKey constants.ContextKey
}

// NewContextProvider creates a new ContextProvider with the specified context key
func NewContextProvider(contextKey constants.ContextKey) *ContextProvider {
	return &ContextProvider{
		contextKey: contextKey,
	}
}

// GetInputData implements InputDataProvider.GetInputData
// It extracts a map[string]any from the context using the configured key
func (p *ContextProvider) GetInputData(ctx context.Context) (map[string]any, error) {
	if p.contextKey == "" {
		return nil, fmt.Errorf("context key is empty")
	}

	// Get data from context using the key
	value := ctx.Value(p.contextKey)
	if value == nil {
		return make(map[string]any), nil
	}

	// Type assert to map[string]any
	inputData, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input data type: expected map[string]any, got %T", value)
	}

	return inputData, nil
}
