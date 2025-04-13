package httpauth

import (
	"context"
	"maps"
	"net/http"
)

// HeaderAuth implements authentication via custom HTTP headers.
// This is a flexible authenticator that can be used for various header-based
// authentication schemes like Bearer tokens, API keys, and custom authentication.
type HeaderAuth struct {
	Headers map[string]string
}

// NewHeaderAuth creates a new HeaderAuth authenticator with the given headers.
// Each key-value pair in the map will be set as a header in the HTTP request.
func NewHeaderAuth(headers map[string]string) *HeaderAuth {
	return &HeaderAuth{
		Headers: maps.Clone(headers),
	}
}

// NewBearerAuth creates a HeaderAuth specifically configured for Bearer token authentication.
// This is a common authentication method for APIs that use JWT or OAuth tokens.
func NewBearerAuth(token string) *HeaderAuth {
	return &HeaderAuth{
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
	}
}

// Authenticate applies header-based authentication to the HTTP request
// by setting all headers from the Headers map.
func (h *HeaderAuth) Authenticate(req *http.Request) error {
	for key, value := range h.Headers {
		req.Header.Set(key, value)
	}
	return nil
}

// AuthenticateWithContext applies header-based authentication with context support.
// This respects context cancellation while applying authentication.
func (h *HeaderAuth) AuthenticateWithContext(ctx context.Context, req *http.Request) error {
	return applyAuthWithContext(ctx, req, h.Authenticate)
}

// Name returns the name of the authentication method.
func (h *HeaderAuth) Name() string {
	return "Header"
}
