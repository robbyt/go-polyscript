// Package loader provides implementations of the Loader interface for various source types.
//
// This file implements the HTTP loader, which allows fetching scripts from HTTP/HTTPS URLs.
// The loader supports various authentication methods via the httpauth package:
//   - No authentication: Use loader.WithNoAuth()
//   - Basic authentication: Use loader.WithBasicAuth(username, password)
//   - Bearer token: Use loader.WithBearerAuth(token)
//   - Custom headers: Use loader.WithHeaderAuth(headers)
//
// The loader also supports context-based operations for timeout and cancellation control.
package loader

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/robbyt/go-polyscript/execution/script/loader/httpauth"
	"github.com/robbyt/go-polyscript/internal/helpers"
)

// HTTPOptions contains configuration options for HTTP loader.
// Use DefaultHTTPOptions() to get sensible defaults, then modify as needed.
//
// Example:
//
//	options := loader.DefaultHTTPOptions()
//	options.Timeout = 10 * time.Second
//	options = options.WithBasicAuth("user", "pass")
type HTTPOptions struct {
	// Timeout specifies a time limit for requests made by this Client
	// Default is 30 seconds if using DefaultHTTPOptions()
	Timeout time.Duration

	// TLSConfig specifies the TLS configuration to use
	// This is optional and allows for advanced TLS configuration
	TLSConfig *tls.Config

	// InsecureSkipVerify skips TLS certificate verification when set to true
	// Default is false (certificates are verified) in DefaultHTTPOptions()
	// Warning: Setting to true reduces security and should only be done in test environments
	InsecureSkipVerify bool

	// Authentication to use for HTTP requests
	Authenticator httpauth.Authenticator

	// Headers for additional headers not related to authentication
	Headers map[string]string
}

// DefaultHTTPOptions returns default options for HTTP loader.
// This provides a sensible starting point that you can then customize.
//
// Default values:
// - Timeout: 30 seconds
// - InsecureSkipVerify: false (certificate validation enabled)
// - Authenticator: NoAuth (no authentication)
// - Headers: empty map (initialized, ready for values)
func DefaultHTTPOptions() *HTTPOptions {
	return &HTTPOptions{
		Timeout:            30 * time.Second,
		InsecureSkipVerify: false,
		Authenticator:      httpauth.NewNoAuth(),
		Headers:            make(map[string]string),
	}
}

// WithBasicAuth returns a copy of options with Basic authentication set.
func (o *HTTPOptions) WithBasicAuth(username, password string) *HTTPOptions {
	newOpts := *o
	newOpts.Authenticator = httpauth.NewBasicAuth(username, password)
	return &newOpts
}

// WithBearerAuth returns a copy of options with Bearer token authentication set.
func (o *HTTPOptions) WithBearerAuth(token string) *HTTPOptions {
	newOpts := *o
	newOpts.Authenticator = httpauth.NewBearerAuth(token)
	return &newOpts
}

// WithHeaderAuth returns a copy of options with custom header authentication set.
func (o *HTTPOptions) WithHeaderAuth(headers map[string]string) *HTTPOptions {
	newOpts := *o
	newOpts.Authenticator = httpauth.NewHeaderAuth(headers)
	return &newOpts
}

// WithNoAuth returns a copy of options with no authentication set.
func (o *HTTPOptions) WithNoAuth() *HTTPOptions {
	newOpts := *o
	newOpts.Authenticator = httpauth.NewNoAuth()
	return &newOpts
}

// WithTimeout returns a copy of options with the specified timeout.
func (o *HTTPOptions) WithTimeout(timeout time.Duration) *HTTPOptions {
	newOpts := *o
	newOpts.Timeout = timeout
	return &newOpts
}

type httpRequester interface {
	Do(req *http.Request) (*http.Response, error)
}

// FromHTTP implements a loader for HTTP/HTTPS URLs.
// It allows loading scripts from remote web servers with various authentication options.
type FromHTTP struct {
	url       string
	sourceURL *url.URL
	options   *HTTPOptions
	client    httpRequester
}

// NewFromHTTP creates a new HTTP loader with the given URL and default options.
// This is the simplest way to create an HTTP loader when you don't need custom configuration.
//
// Example:
//
//	loader, err := loader.NewFromHTTP("https://example.com/script.js")
//	if err != nil {
//	    return err
//	}
//
// See also NewFromHTTPWithOptions for more customization options.
func NewFromHTTP(rawURL string) (*FromHTTP, error) {
	return NewFromHTTPWithOptions(rawURL, DefaultHTTPOptions())
}

// NewFromHTTPWithOptions creates a new HTTP loader with the given URL and custom options.
// Use this when you need authentication, custom timeouts, or other HTTP configuration.
//
// Examples:
//
//	// With basic auth
//	options := loader.DefaultHTTPOptions().WithBasicAuth("user", "pass")
//	loader, err := loader.NewFromHTTPWithOptions("https://example.com/script.js", options)
//
//	// With bearer token
//	options := loader.DefaultHTTPOptions().WithBearerAuth("token123")
//	loader, err := loader.NewFromHTTPWithOptions("https://example.com/script.js", options)
//
//	// With custom timeout
//	options := loader.DefaultHTTPOptions().WithTimeout(10 * time.Second)
//	loader, err := loader.NewFromHTTPWithOptions("https://example.com/script.js", options)
func NewFromHTTPWithOptions(rawURL string, options *HTTPOptions) (*FromHTTP, error) {
	sourceURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse URL: %w", err)
	}

	if sourceURL.Scheme != "http" && sourceURL.Scheme != "https" {
		return nil, fmt.Errorf("%w: %s", ErrSchemeUnsupported, rawURL)
	}

	// Create HTTP client with specified options
	client := &http.Client{
		Timeout: options.Timeout,
	}

	// Configure TLS if needed
	if options.InsecureSkipVerify || options.TLSConfig != nil {
		// Start with default transport
		transport := http.DefaultTransport.(*http.Transport).Clone()

		// Configure TLS
		tlsConfig := transport.TLSClientConfig
		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		}

		// Apply custom config or skip verify flag
		if options.TLSConfig != nil {
			tlsConfig = options.TLSConfig
		} else if options.InsecureSkipVerify {
			tlsConfig.InsecureSkipVerify = true
		}

		transport.TLSClientConfig = tlsConfig
		client.Transport = transport
	}

	return &FromHTTP{
		url:       rawURL,
		sourceURL: sourceURL,
		options:   options,
		client:    client,
	}, nil
}

// GetReader returns a reader for the HTTP content.
// This method is part of the Loader interface and is used internally by
// the polyscript system to fetch the script content.
//
// The returned io.ReadCloser must be closed by the caller when done.
// HTTP errors are handled and converted to appropriate error types.
func (l *FromHTTP) GetReader() (io.ReadCloser, error) {
	return l.GetReaderWithContext(context.Background())
}

// GetReaderWithContext returns a reader for the HTTP content with context support.
// This allows for request cancellation and timeouts via context.
//
// The returned io.ReadCloser must be closed by the caller when done.
// HTTP errors are handled and converted to appropriate error types.
func (l *FromHTTP) GetReaderWithContext(ctx context.Context) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply authentication
	if l.options.Authenticator != nil {
		if err := l.options.Authenticator.AuthenticateWithContext(ctx, req); err != nil {
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Add any additional headers
	for key, value := range l.options.Headers {
		req.Header.Set(key, value)
	}

	// Set a default User-Agent if not specified
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "go-polyscript/http-loader")
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if err := resp.Body.Close(); err != nil {
			slog.Default().Debug("Failed to close response body", "error", err)
		}
		return nil, fmt.Errorf(
			"%w: HTTP %d - %s",
			ErrScriptNotAvailable,
			resp.StatusCode,
			resp.Status,
		)
	}

	return resp.Body, nil
}

// GetSourceURL returns the source URL.
// This method is part of the Loader interface and identifies the source location.
func (l *FromHTTP) GetSourceURL() *url.URL {
	return l.sourceURL
}

// String returns a string representation of the HTTP loader.
// This is useful for debugging and logging.
func (l *FromHTTP) String() string {
	var chksum string
	noChkSum := fmt.Sprintf("loader.FromHTTP{URL: %s}", l.url)

	if l.sourceURL != nil {
		reader, err := l.GetReader()
		if err != nil {
			return noChkSum
		}
		defer func() {
			if err := reader.Close(); err != nil {
				slog.Default().Debug("Failed to close reader in String() method", "error", err)
			}
		}()

		chksum, err = helpers.SHA256Reader(reader)
		if err != nil {
			return noChkSum
		}

		chksum = chksum[:8]
	}

	if chksum == "" {
		return noChkSum
	}

	return fmt.Sprintf("loader.FromHTTP{URL: %s, SHA256: %s}", l.url, chksum)
}
