package httpauth

import (
	"context"
	"net/http"
)

// NoAuth represents a no-authentication strategy.
// This authenticator performs no operations and is useful as a default
// or for explicitly indicating that no authentication is required.
type NoAuth struct{}

// NewNoAuth creates a new NoAuth authenticator instance.
func NewNoAuth() *NoAuth {
	return &NoAuth{}
}

// Authenticate does nothing as no authentication is needed.
// This method satisfies the Authenticator interface but performs no operations.
func (n *NoAuth) Authenticate(req *http.Request) error {
	// No action needed for no authentication
	return nil
}

// AuthenticateWithContext does nothing as no authentication is needed,
// but respects context cancellation.
func (n *NoAuth) AuthenticateWithContext(ctx context.Context, req *http.Request) error {
	return applyAuthWithContext(ctx, req, n.Authenticate)
}

// Name returns the name of the authentication method.
func (n *NoAuth) Name() string {
	return "None"
}
