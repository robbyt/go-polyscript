package data

import (
	"context"
)

// Provider is an interface for retrieving data for script evaluation
// This is the new interface that should be used going forward with the simplified evaluator pattern
type Provider interface {
	// GetData retrieves the data map from the given context
	// Returns a map of string keys to arbitrary values that will be passed to the script
	// If an error occurs during data retrieval, it will be returned
	GetData(ctx context.Context) (map[string]any, error)
}
