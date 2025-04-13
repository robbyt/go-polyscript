package httpauth

import (
	"context"
	"net/http"
)

// BasicAuth implements HTTP Basic Authentication according to RFC 7617.
// It adds an Authorization header with the format: "Basic base64(username:password)".
type BasicAuth struct {
	Username string
	Password string
}

// NewBasicAuth creates a new BasicAuth authenticator with the given credentials.
// If username is empty, this authenticator will do nothing when called.
func NewBasicAuth(username, password string) *BasicAuth {
	return &BasicAuth{
		Username: username,
		Password: password,
	}
}

// Authenticate applies basic authentication to the HTTP request by
// setting the Authorization header using the standard Basic auth format.
// If the username is empty, no authentication is applied.
func (b *BasicAuth) Authenticate(req *http.Request) error {
	if b.Username != "" {
		req.SetBasicAuth(b.Username, b.Password)
	}
	return nil
}

// AuthenticateWithContext applies basic authentication with context support.
// This respects context cancellation while applying authentication.
func (b *BasicAuth) AuthenticateWithContext(ctx context.Context, req *http.Request) error {
	return applyAuthWithContext(ctx, req, b.Authenticate)
}

// Name returns the name of the authentication method.
func (b *BasicAuth) Name() string {
	return "Basic"
}
