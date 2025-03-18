package loader

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewFromHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		url           string
		expectError   bool
		errorContains string
	}{
		{
			name:        "Valid HTTPS URL",
			url:         "https://example.com/script.js",
			expectError: false,
		},
		{
			name:        "Valid HTTP URL",
			url:         "http://example.com/script.js",
			expectError: false,
		},
		{
			name:          "Invalid URL scheme",
			url:           "file:///path/to/script.js",
			expectError:   true,
			errorContains: "unsupported scheme",
		},
		{
			name:          "Invalid URL format",
			url:           "://invalid-url",
			expectError:   true,
			errorContains: "unable to parse URL",
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loader, err := NewFromHTTP(tt.url)
			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, loader)
			require.Equal(t, tt.url, loader.url)
			require.NotNil(t, loader.sourceURL)
			require.Equal(t, tt.url, loader.sourceURL.String())
			require.NotNil(t, loader.client)
			require.NotNil(t, loader.options)
		})
	}
}

func TestNewFromHTTPWithOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		url            string
		options        *HTTPOptions
		validateOption func(t *testing.T, loader *FromHTTP)
		expectError    bool
		errorContains  string
	}{
		{
			name: "Custom timeout",
			url:  "https://example.com/script.js",
			options: &HTTPOptions{
				Timeout: 60 * time.Second,
			},
			validateOption: func(t *testing.T, loader *FromHTTP) {
				require.Equal(t, 60*time.Second, loader.client.Timeout)
			},
		},
		{
			name: "Basic auth",
			url:  "https://example.com/script.js",
			options: &HTTPOptions{
				AuthType: BasicAuth,
				Username: "user",
				Password: "pass",
			},
			validateOption: func(t *testing.T, loader *FromHTTP) {
				require.Equal(t, BasicAuth, loader.options.AuthType)
				require.Equal(t, "user", loader.options.Username)
				require.Equal(t, "pass", loader.options.Password)
			},
		},
		{
			name: "Custom headers",
			url:  "https://example.com/script.js",
			options: &HTTPOptions{
				AuthType: HeaderAuth,
				Headers: map[string]string{
					"Authorization": "Bearer token",
					"User-Agent":    "Test-Agent",
				},
			},
			validateOption: func(t *testing.T, loader *FromHTTP) {
				require.Equal(t, HeaderAuth, loader.options.AuthType)
				require.Equal(t, "Bearer token", loader.options.Headers["Authorization"])
				require.Equal(t, "Test-Agent", loader.options.Headers["User-Agent"])
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Use either provided options or default options
			options := tt.options
			if options == nil {
				options = DefaultHTTPOptions()
			} else {
				// Fill in any unset fields with defaults
				defaultOptions := DefaultHTTPOptions()
				if options.Timeout == 0 {
					options.Timeout = defaultOptions.Timeout
				}
				if options.Headers == nil {
					options.Headers = defaultOptions.Headers
				}
			}

			loader, err := NewFromHTTPWithOptions(tt.url, options)
			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, loader)
			require.Equal(t, tt.url, loader.url)
			require.NotNil(t, loader.sourceURL)
			require.Equal(t, tt.url, loader.sourceURL.String())
			require.NotNil(t, loader.client)
			require.NotNil(t, loader.options)

			if tt.validateOption != nil {
				tt.validateOption(t, loader)
			}
		})
	}
}

func TestFromHTTPGetReader(t *testing.T) {
	// Not running in parallel to ensure the server is available for all subtests

	const testScript = `function test() { return "Hello, World!"; }`

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Test basic auth
		if path == "/auth" {
			username, password, ok := r.BasicAuth()
			if !ok || username != "user" || password != "pass" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, err := w.Write([]byte(testScript))
			if err != nil {
				t.Logf("Failed to write response: %v", err)
			}
			return
		}

		// Test header auth
		if path == "/header-auth" {
			token := r.Header.Get("Authorization")
			if token != "Bearer test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, err := w.Write([]byte(testScript))
			if err != nil {
				t.Logf("Failed to write response: %v", err)
			}
			return
		}

		// Test error response
		if path == "/error" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Default response
		_, err := w.Write([]byte(testScript))
		if err != nil {
			t.Logf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	tests := []struct {
		name          string
		url           string
		options       *HTTPOptions
		expectError   bool
		errorContains string
		validateBody  bool
	}{
		{
			name:         "Success - Default",
			url:          server.URL + "/script.js",
			validateBody: true,
		},
		{
			name: "Success - Basic Auth",
			url:  server.URL + "/auth",
			options: &HTTPOptions{
				AuthType: BasicAuth,
				Username: "user",
				Password: "pass",
				Timeout:  5 * time.Second,
			},
			validateBody: true,
		},
		{
			name: "Success - Header Auth",
			url:  server.URL + "/header-auth",
			options: &HTTPOptions{
				AuthType: HeaderAuth,
				Headers: map[string]string{
					"Authorization": "Bearer test-token",
				},
				Timeout: 5 * time.Second,
			},
			validateBody: true,
		},
		{
			name: "Failure - Basic Auth Invalid Credentials",
			url:  server.URL + "/auth",
			options: &HTTPOptions{
				AuthType: BasicAuth,
				Username: "wrong",
				Password: "wrong",
				Timeout:  5 * time.Second,
			},
			expectError:   true,
			errorContains: "HTTP 401",
		},
		{
			name: "Failure - Header Auth Invalid Token",
			url:  server.URL + "/header-auth",
			options: &HTTPOptions{
				AuthType: HeaderAuth,
				Headers: map[string]string{
					"Authorization": "Bearer wrong-token",
				},
				Timeout: 5 * time.Second,
			},
			expectError:   true,
			errorContains: "HTTP 401",
		},
		{
			name:          "Failure - Not Found",
			url:           server.URL + "/error",
			expectError:   true,
			errorContains: "HTTP 404",
		},
		{
			name:          "Failure - Invalid URL",
			url:           "http://invalid-domain-that-should-not-exist.example",
			expectError:   true,
			errorContains: "failed to execute HTTP request",
		},
		{
			name: "Failure - Digest Auth Not Implemented",
			url:  server.URL + "/script.js",
			options: &HTTPOptions{
				AuthType: DigestAuth,
				Timeout:  5 * time.Second,
			},
			expectError:   true,
			errorContains: "digest authentication not implemented",
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			// Not running subtests in parallel to avoid server closing issues

			var loader *FromHTTP
			var err error

			if tt.options != nil {
				loader, err = NewFromHTTPWithOptions(tt.url, tt.options)
			} else {
				loader, err = NewFromHTTP(tt.url)
			}
			require.NoError(t, err, "Failed to create HTTP loader")

			reader, err := loader.GetReader()
			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, reader)
			defer reader.Close()

			if tt.validateBody {
				content, err := io.ReadAll(reader)
				require.NoError(t, err)
				require.Equal(t, testScript, string(content))
			}
		})
	}
}

func TestFromHTTPString(t *testing.T) {
	// Not running in parallel to avoid test server issues

	// Create test server that returns a predictable response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("test script content"))
		if err != nil {
			t.Logf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	loader, err := NewFromHTTP(server.URL)
	require.NoError(t, err)

	str := loader.String()
	require.Contains(t, str, "loader.FromHTTP{URL:")
	require.Contains(t, str, server.URL)

	// Test error case for String method
	invalidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close connection immediately to cause error
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Close()
	}))
	defer invalidServer.Close()

	invalidLoader, err := NewFromHTTP(invalidServer.URL)
	require.NoError(t, err)

	str = invalidLoader.String()
	require.Contains(t, str, "loader.FromHTTP{URL:")
	require.Contains(t, str, invalidServer.URL)
	require.NotContains(t, str, "SHA256")

	// Test String() with nonexistent server
	nonExistentLoader, err := NewFromHTTP("http://nonexistent-server.example")
	require.NoError(t, err)

	str = nonExistentLoader.String()
	require.Contains(t, str, "loader.FromHTTP{URL:")
	require.Contains(t, str, "nonexistent-server.example")
	require.NotContains(t, str, "SHA256")
}

func TestDefaultHTTPOptions(t *testing.T) {
	t.Parallel()

	options := DefaultHTTPOptions()
	require.NotNil(t, options)
	require.Equal(t, 30*time.Second, options.Timeout)
	require.False(t, options.InsecureSkipVerify)
	require.Equal(t, NoAuth, options.AuthType)
	require.NotNil(t, options.Headers)
	require.Empty(t, options.Headers)
}
