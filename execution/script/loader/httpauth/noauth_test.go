package httpauth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNoAuth(t *testing.T) {
	t.Parallel()

	// Create the authenticator
	auth := NewNoAuth()

	// Check the name
	require.Equal(t, "None", auth.Name())

	// Run various authentication scenarios
	tests := []struct {
		name        string
		setup       func(t *testing.T) (*http.Request, context.Context, context.CancelFunc)
		expectError bool
		verifyReq   func(t *testing.T, req *http.Request)
	}{
		{
			name: "Basic authentication",
			setup: func(t *testing.T) (*http.Request, context.Context, context.CancelFunc) {
				req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
				require.NoError(t, err)
				return req, nil, nil
			},
			verifyReq: func(t *testing.T, req *http.Request) {
				// No-auth shouldn't add any headers
				require.Empty(t, req.Header.Get("Authorization"))
			},
		},
		{
			name: "With context authentication",
			setup: func(t *testing.T) (*http.Request, context.Context, context.CancelFunc) {
				req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
				require.NoError(t, err)
				ctx := context.Background()
				return req, ctx, nil
			},
			verifyReq: func(t *testing.T, req *http.Request) {
				require.Empty(t, req.Header.Get("Authorization"))
			},
		},
		{
			name: "With cancelled context",
			setup: func(t *testing.T) (*http.Request, context.Context, context.CancelFunc) {
				req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
				require.NoError(t, err)
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return req, ctx, nil
			},
			expectError: true,
		},
		{
			name: "With timeout context",
			setup: func(t *testing.T) (*http.Request, context.Context, context.CancelFunc) {
				req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
				require.NoError(t, err)
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

			req, ctx, cancel := tt.setup(t)
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
