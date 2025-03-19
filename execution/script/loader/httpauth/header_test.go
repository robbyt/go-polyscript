package httpauth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHeaderAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		factory     func() Authenticator
		setup       func() (*http.Request, context.Context, context.CancelFunc)
		expectError bool
		verifyReq   func(t *testing.T, req *http.Request)
	}{
		{
			name: "Multiple custom headers",
			factory: func() Authenticator {
				return NewHeaderAuth(map[string]string{
					"Authorization": "Bearer token123",
					"X-API-Key":     "secret-key",
					"X-Custom":      "value",
				})
			},
			setup: func() (*http.Request, context.Context, context.CancelFunc) {
				req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
				return req, nil, nil
			},
			verifyReq: func(t *testing.T, req *http.Request) {
				require.Equal(t, "Bearer token123", req.Header.Get("Authorization"))
				require.Equal(t, "secret-key", req.Header.Get("X-API-Key"))
				require.Equal(t, "value", req.Header.Get("X-Custom"))
			},
		},
		{
			name: "Empty headers map",
			factory: func() Authenticator {
				return NewHeaderAuth(map[string]string{})
			},
			setup: func() (*http.Request, context.Context, context.CancelFunc) {
				req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
				return req, nil, nil
			},
			verifyReq: func(t *testing.T, req *http.Request) {
				require.Empty(t, req.Header.Get("Authorization"))
			},
		},
		{
			name: "Nil headers map",
			factory: func() Authenticator {
				return NewHeaderAuth(nil)
			},
			setup: func() (*http.Request, context.Context, context.CancelFunc) {
				req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
				return req, nil, nil
			},
			verifyReq: func(t *testing.T, req *http.Request) {
				require.Empty(t, req.Header.Get("Authorization"))
			},
		},
		{
			name: "Bearer token helper",
			factory: func() Authenticator {
				return NewBearerAuth("my-test-token")
			},
			setup: func() (*http.Request, context.Context, context.CancelFunc) {
				req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
				return req, nil, nil
			},
			verifyReq: func(t *testing.T, req *http.Request) {
				require.Equal(t, "Bearer my-test-token", req.Header.Get("Authorization"))
			},
		},
		{
			name: "With context",
			factory: func() Authenticator {
				return NewHeaderAuth(map[string]string{
					"Authorization": "Bearer token123",
				})
			},
			setup: func() (*http.Request, context.Context, context.CancelFunc) {
				req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
				ctx := context.Background()
				return req, ctx, nil
			},
			verifyReq: func(t *testing.T, req *http.Request) {
				require.Equal(t, "Bearer token123", req.Header.Get("Authorization"))
			},
		},
		{
			name: "With cancelled context",
			factory: func() Authenticator {
				return NewHeaderAuth(map[string]string{
					"Authorization": "Bearer token123",
				})
			},
			setup: func() (*http.Request, context.Context, context.CancelFunc) {
				req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return req, ctx, nil
			},
			expectError: true,
		},
		{
			name: "With timeout context",
			factory: func() Authenticator {
				return NewHeaderAuth(map[string]string{
					"Authorization": "Bearer token123",
				})
			},
			setup: func() (*http.Request, context.Context, context.CancelFunc) {
				req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				time.Sleep(5 * time.Millisecond) // Ensure the timeout occurs
				return req, ctx, cancel
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			auth := tt.factory()
			require.Equal(t, "Header", auth.Name())

			req, ctx, cancel := tt.setup()
			if cancel != nil {
				defer cancel()
			}

			var err error
			if ctx != nil {
				err = auth.AuthenticateWithContext(ctx, req)
			} else {
				err = auth.Authenticate(req)
			}

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.verifyReq != nil {
				tt.verifyReq(t, req)
			}
		})
	}
}

// Test that maps.Clone is properly cloning the headers map
func TestHeaderAuthCloning(t *testing.T) {
	t.Parallel()

	// Create original map
	originalHeaders := map[string]string{
		"X-Test": "value",
	}

	// Create authenticator
	auth := NewHeaderAuth(originalHeaders)

	// Modify original map after creation
	originalHeaders["X-Test"] = "modified"
	originalHeaders["X-New"] = "added"

	// Create a request
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)

	// Apply auth
	err := auth.Authenticate(req)
	require.NoError(t, err)

	// The authenticator should have used its own internal copy, not the modified one
	require.Equal(t, "value", req.Header.Get("X-Test"))
	require.Empty(t, req.Header.Get("X-New"))
}
