// Package loader provides implementations of the Loader interface for various source types.
package loader

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/robbyt/go-polyscript/internal/helpers"
)

// HTTPAuthType defines the authentication type for HTTP loader.
// Different authentication methods can be used when fetching scripts from HTTP sources.
type HTTPAuthType string

const (
	// NoAuth represents no authentication
	NoAuth HTTPAuthType = "none"

	// BasicAuth represents HTTP Basic Authentication
	// When using this auth type, provide Username and Password fields in HTTPOptions.
	// Example:
	//   options.AuthType = loader.BasicAuth
	//   options.Username = "user"
	//   options.Password = "pass"
	BasicAuth HTTPAuthType = "basic"

	// HeaderAuth represents authentication via custom headers
	// When using this auth type, provide Headers map in HTTPOptions.
	// Example for Bearer token:
	//   options.AuthType = loader.HeaderAuth
	//   options.Headers["Authorization"] = "Bearer your-token-here"
	HeaderAuth HTTPAuthType = "header"

	// DigestAuth represents HTTP Digest Authentication
	// Note: This auth type is not currently implemented and will return an error if used.
	DigestAuth HTTPAuthType = "digest"
)

// HTTPOptions contains configuration options for HTTP loader.
// Use DefaultHTTPOptions() to get sensible defaults, then modify as needed.
//
// Example:
//
//	options := loader.DefaultHTTPOptions()
//	options.Timeout = 10 * time.Second
//	options.AuthType = loader.BasicAuth
//	options.Username = "user"
//	options.Password = "pass"
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

	// AuthType specifies the authentication type to use
	// Default is NoAuth in DefaultHTTPOptions()
	AuthType HTTPAuthType

	// Username and Password for BasicAuth
	// Only used when AuthType is set to BasicAuth
	Username string
	Password string

	// Headers for HeaderAuth or additional headers
	// When AuthType is HeaderAuth, add authentication information here
	// Example: headers["Authorization"] = "Bearer token123"
	Headers map[string]string
}

// DefaultHTTPOptions returns default options for HTTP loader.
// This provides a sensible starting point that you can then customize.
//
// Default values:
// - Timeout: 30 seconds
// - InsecureSkipVerify: false (certificate validation enabled)
// - AuthType: NoAuth (no authentication)
// - Headers: empty map (initialized, ready for values)
func DefaultHTTPOptions() *HTTPOptions {
	return &HTTPOptions{
		Timeout:            30 * time.Second,
		InsecureSkipVerify: false,
		AuthType:           NoAuth,
		Headers:            make(map[string]string),
	}
}

// FromHTTP implements a loader for HTTP/HTTPS URLs.
// It allows loading scripts from remote web servers with various authentication options.
type FromHTTP struct {
	url       string
	sourceURL *url.URL
	options   *HTTPOptions
	client    *http.Client
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
//	options := loader.DefaultHTTPOptions()
//	options.AuthType = loader.BasicAuth
//	options.Username = "user"
//	options.Password = "pass"
//	loader, err := loader.NewFromHTTPWithOptions("https://example.com/script.js", options)
//
//	// With bearer token
//	options := loader.DefaultHTTPOptions()
//	options.AuthType = loader.HeaderAuth
//	options.Headers["Authorization"] = "Bearer token123"
//	loader, err := loader.NewFromHTTPWithOptions("https://example.com/script.js", options)
//
//	// With custom timeout
//	options := loader.DefaultHTTPOptions()
//	options.Timeout = 10 * time.Second
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
		transport := http.DefaultTransport.(*http.Transport).Clone()
		if options.TLSConfig != nil {
			transport.TLSClientConfig = options.TLSConfig
		} else if options.InsecureSkipVerify {
			transport.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
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
	req, err := http.NewRequest(http.MethodGet, l.url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply authentication based on auth type
	switch l.options.AuthType {
	case BasicAuth:
		if l.options.Username != "" {
			req.SetBasicAuth(l.options.Username, l.options.Password)
		}
	case HeaderAuth:
		// Apply custom headers
		for key, value := range l.options.Headers {
			req.Header.Set(key, value)
		}
	case DigestAuth:
		return nil, fmt.Errorf("digest authentication not implemented")
	}

	// Add any additional headers
	if l.options.AuthType != HeaderAuth {
		for key, value := range l.options.Headers {
			req.Header.Set(key, value)
		}
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
		resp.Body.Close()
		return nil, fmt.Errorf("%w: HTTP %d - %s", ErrScriptNotAvailable, resp.StatusCode, resp.Status)
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
		defer reader.Close()

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
