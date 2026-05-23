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
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/platform/script/loader/httpauth"
)

// DefaultMaxBodySize is the default cap on HTTP-loaded script bodies.
// 10 MiB is large enough for any realistic script while keeping a single
// rogue response from exhausting memory.
const DefaultMaxBodySize int64 = 10 << 20

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

	// MaxBodySize caps the response body in bytes. A zero value is treated
	// as DefaultMaxBodySize; a negative value disables the cap. Bodies
	// larger than the limit cause GetReader to return ErrScriptTooLarge.
	MaxBodySize int64
}

// DefaultHTTPOptions returns default options for HTTP loader.
// This provides a sensible starting point that you can then customize.
//
// Default values:
// - Timeout: 30 seconds
// - InsecureSkipVerify: false (certificate validation enabled)
// - Authenticator: NoAuth (no authentication)
// - Headers: empty map (initialized, ready for values)
// - MaxBodySize: DefaultMaxBodySize (10 MiB)
func DefaultHTTPOptions() *HTTPOptions {
	return &HTTPOptions{
		Timeout:            30 * time.Second,
		InsecureSkipVerify: false,
		Authenticator:      httpauth.NewNoAuth(),
		Headers:            make(map[string]string),
		MaxBodySize:        DefaultMaxBodySize,
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

// WithMaxBodySize returns a copy of options with the response body cap set.
// Pass 0 to fall back to DefaultMaxBodySize, or a negative value to disable
// the cap entirely.
func (o *HTTPOptions) WithMaxBodySize(n int64) *HTTPOptions {
	newOpts := *o
	newOpts.MaxBodySize = n
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

	// sha holds a short (8-char) SHA-256 prefix of the response body,
	// cached on the first successful GetReader call. String() loads it
	// without making a network request. Nil until the first body has
	// been buffered by cappedBody — when MaxBodySize is negative the
	// body is streamed and the cache stays unset.
	sha atomic.Pointer[string]
}

// NewFromHTTP creates a new HTTP loader with the given URL and default options.
// This is the simplest way to create an HTTP loader when you don't need custom configuration.
//
// Example:
//
//	loader, err := loader.NewFromHTTP("https://localhost:8080/script.js")
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
//	loader, err := loader.NewFromHTTPWithOptions("https://localhost:8080/script.js", options)
//
//	// With bearer token
//	options := loader.DefaultHTTPOptions().WithBearerAuth("token123")
//	loader, err := loader.NewFromHTTPWithOptions("https://localhost:8080/script.js", options)
//
//	// With custom timeout
//	options := loader.DefaultHTTPOptions().WithTimeout(10 * time.Second)
//	loader, err := loader.NewFromHTTPWithOptions("https://localhost:8080/script.js", options)
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

// GetReader returns a reader for the HTTP content. The ctx flows into
// the HTTP request and the authenticator so a cancelled ctx aborts the
// fetch.
//
// The returned io.ReadCloser must be closed by the caller when done.
// HTTP errors are handled and converted to appropriate error types.
func (l *FromHTTP) GetReader(ctx context.Context) (io.ReadCloser, error) {
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

	return l.cappedBody(resp)
}

// cappedBody enforces HTTPOptions.MaxBodySize on the response. A zero
// value falls back to DefaultMaxBodySize; a negative value returns the
// body unwrapped. Otherwise the body is eagerly read into memory and
// returned as a NopCloser; ErrScriptTooLarge is returned when it
// exceeds the limit.
func (l *FromHTTP) cappedBody(resp *http.Response) (io.ReadCloser, error) {
	limit := l.options.MaxBodySize
	if limit == 0 {
		limit = DefaultMaxBodySize
	}
	if limit < 0 {
		return resp.Body, nil
	}

	// Read up to limit+1 bytes so a body of exactly limit bytes passes
	// while one byte more trips the overflow check. Guard against
	// MaxInt64 overflow: if limit is already at the int64 ceiling no
	// realistic body can exceed it, so we read up to limit and the
	// overflow branch is unreachable.
	readLimit := limit
	if readLimit < math.MaxInt64 {
		readLimit++
	}
	buf, err := io.ReadAll(io.LimitReader(resp.Body, readLimit))
	if closeErr := resp.Body.Close(); closeErr != nil {
		slog.Default().Debug("Failed to close response body", "error", closeErr)
	}
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if int64(len(buf)) > limit {
		return nil, fmt.Errorf("%w: %s exceeds %d bytes", ErrScriptTooLarge, l.url, limit)
	}

	// Cache the short SHA prefix from the first successful read so
	// String() can include it without a network round-trip. CAS-on-nil
	// keeps the first writer's value if multiple goroutines race here.
	if l.sha.Load() == nil {
		sum := helpers.SHA256Bytes(buf)
		if len(sum) >= 8 {
			short := sum[:8]
			l.sha.CompareAndSwap(nil, &short)
		}
	}

	return io.NopCloser(bytes.NewReader(buf)), nil
}

// GetSourceURL returns the source URL.
// This method is part of the Loader interface and identifies the source location.
func (l *FromHTTP) GetSourceURL() *url.URL {
	return l.sourceURL
}

// String returns a string representation of the HTTP loader, suitable
// for debugging and logging. The returned value is cheap to compute:
// it never performs a network request.
//
// The "SHA256: <prefix>" form is included only after the body has been
// fetched at least once via [FromHTTP.GetReader] and the response was
// buffered through [FromHTTP.cappedBody]. Until then — or when [HTTPOptions.MaxBodySize]
// is negative (which disables buffering) — String returns the
// no-checksum form.
func (l *FromHTTP) String() string {
	if p := l.sha.Load(); p != nil {
		return fmt.Sprintf("loader.FromHTTP{URL: %s, SHA256: %s}", l.url, *p)
	}
	return fmt.Sprintf("loader.FromHTTP{URL: %s}", l.url)
}
