package data

import (
	"context"
)

// InputDataProvider is an interface for retrieving input data for script evaluation
// This allows different strategies for data retrieval to be implemented and used
// interchangeably with the script evaluators.
type InputDataProvider interface {
	// GetInputData retrieves the input data map from the given context
	// Returns a map of string keys to arbitrary values that will be passed to the script
	// If an error occurs during data retrieval, it will be returned
	GetInputData(ctx context.Context) (map[string]any, error)
}
