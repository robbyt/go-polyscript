// Package httpauth provides authentication strategies for HTTP requests in go-polyscript.
package httpauth

import (
	"context"
	"net/http"
)

// Authenticator defines the interface for HTTP authentication strategies.
type Authenticator interface {
	// Authenticate applies authentication to the HTTP request.
	// It should modify the request in-place to add authentication details.
	Authenticate(req *http.Request) error

	// AuthenticateWithContext applies authentication with context.
	// This allows for context cancellation during authentication.
	AuthenticateWithContext(ctx context.Context, req *http.Request) error

	// Name returns a descriptive name of the authentication method.
	Name() string
}

// applyAuthWithContext is a helper function that applies authentication
// while respecting context cancellation. It first checks if the context
// is already canceled, and if not, it attaches the context to the request
// and calls the provided authentication function.
//
// This function is intended for use by Authenticator implementations to
// provide a consistent way to handle context in AuthenticateWithContext methods.
//
// Parameters:
//   - ctx: The context for cancellation and timeout control
//   - req: The HTTP request to authenticate
//   - authFn: The function that performs the actual authentication
//
// Returns an error if the context is already canceled or if the authentication fails.
func applyAuthWithContext(
	ctx context.Context,
	req *http.Request,
	authFn func(*http.Request) error,
) error {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Context is still valid, continue with authentication
	}

	// Apply the authentication using the request with context
	reqWithCtx := req.WithContext(ctx)
	return authFn(reqWithCtx)
}
