package helpers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"log/slog"
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

// newHttpRequestWrapper converts an http.Request to an httpRequest struct.
func newHttpRequestWrapper(r *http.Request) (*httpRequestWrapper, error) {
	if r == nil {
		return nil, errors.New("request is nil")
	}

	// Validate URL if present
	if r.URL != nil {
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
	reqStruct, err := newHttpRequestWrapper(r)
	if err != nil {
		slog.Error("Failed to transform http.Request to httpRequest struct", "error", err)
		return nil, err
	}

	if reqStruct == nil {
		slog.Error("Failed to transform http.Request to httpRequest struct", "error", "reqStruct is nil")
		return nil, err
	}

	// Convert httpRequest struct to map[string]any
	return reqStruct.toMap(), nil
}
