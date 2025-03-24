package helpers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// httpRequestWrapper is a struct that mirrors the http.Request fields we are interested in.
type httpRequestWrapper struct {
	Method        string
	URL           *url.URL
	Proto         string
	Headers       map[string][]string
	Body          string
	ContentLength int64
	Host          string
	RemoteAddr    string
	QueryParams   map[string][]string
}

// newHTTPRequestWrapper converts an http.Request to an httpRequest struct.
func newHTTPRequestWrapper(r *http.Request) (*httpRequestWrapper, error) {
	if r == nil {
		return nil, errors.New("request is nil")
	}

	// Ensure and validate the URL
	if r.URL == nil {
		// If URL is nil, provide a default one
		r.URL = &url.URL{Path: "/"}
	} else {
		// If URL is not nil, validate it
		if _, err := url.Parse(r.URL.String()); err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}
	}

	reqStruct := &httpRequestWrapper{
		Method:        r.Method,
		URL:           r.URL,
		Proto:         r.Proto,
		ContentLength: r.ContentLength,
		Host:          r.Host,
		RemoteAddr:    r.RemoteAddr,
		Headers:       make(map[string][]string),
		QueryParams:   make(map[string][]string),
	}

	// Copy headers if present
	if r.Header != nil {
		for k, v := range r.Header {
			reqStruct.Headers[k] = v
		}
	}

	// Read and set the body
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		reqStruct.Body = string(bodyBytes)

		// Reset the body to allow further reads
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// Copy query parameters if URL is present
	if r.URL != nil {
		query := r.URL.Query()
		for k, v := range query {
			reqStruct.QueryParams[k] = v
		}
	}

	return reqStruct, nil
}

// toMap converts an httpRequest struct to a map[string]any.
func (h *httpRequestWrapper) toMap() map[string]any {
	return map[string]any{
		"Method":        h.Method,
		"URL":           h.URL,
		"URL_String":    h.URL.String(),
		"URL_Host":      h.URL.Host,
		"URL_Scheme":    h.URL.Scheme,
		"URL_Path":      h.URL.Path,
		"Proto":         h.Proto,
		"Headers":       h.Headers,
		"Body":          h.Body,
		"ContentLength": h.ContentLength,
		"Host":          h.Host,
		"RemoteAddr":    h.RemoteAddr,
		"QueryParams":   h.QueryParams,
	}
}

// RequestToMap converts an http.Request to a map[string]any using the httpRequest struct as an intermediary.
func RequestToMap(r *http.Request) (map[string]any, error) {
	// Transform http.Request to httpRequest struct
	reqStruct, err := newHTTPRequestWrapper(r)
	if err != nil {
		return nil, fmt.Errorf("failed to transform http.Request to httpRequest struct: %w", err)
	}

	if reqStruct == nil {
		return nil, errors.New("failed to transform http.Request to httpRequest struct: result is nil")
	}

	// Convert httpRequest struct to map[string]any
	return reqStruct.toMap(), nil
}
