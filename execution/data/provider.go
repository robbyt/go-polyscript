package data

import (
	"context"
)

// Provider defines the interface for accessing runtime data for script execution.
type Provider interface {
	// GetData retrieves script data using the provided context.
	GetData(ctx context.Context) (map[string]any, error)

	// AddDataToContext adds data to the execution context.
	AddDataToContext(ctx context.Context, data ...any) (context.Context, error)
}
