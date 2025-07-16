package loader

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/robbyt/go-polyscript/internal/helpers"
)

// FromString implements the Loader interface for string content.
type FromString struct {
	content   string
	sourceURL *url.URL
}

// NewFromString creates a new loader from string content.
// The content is trimmed of whitespace and must be non-empty.
func NewFromString(content string) (*FromString, error) {
	content = strings.TrimSpace(content)

	if content == "" {
		return nil, fmt.Errorf("%w: content is empty", ErrScriptNotAvailable)
	}

	// Use a more complete URL with a unique identifier
	u, err := url.Parse("string://inline/" + helpers.SHA256(content)[:8])
	if err != nil {
		return nil, fmt.Errorf("failed to create source URL: %w", err)
	}

	return &FromString{
		content:   content,
		sourceURL: u,
	}, nil
}

// NewFromStringBase64 attempts to base64 decode the input string.
// If decoding fails, it falls back to using the original string directly.
func NewFromStringBase64(content string) (Loader, error) {
	content = strings.TrimSpace(content)

	if content == "" {
		return nil, fmt.Errorf("%w: content is empty", ErrScriptNotAvailable)
	}

	if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
		// return using the NewFromBytes function so the decoded content isn't converted back to a string
		// which could cause issues with special characters.
		return NewFromBytes(decoded)
	}

	// Base64 decoding failed, use the original string
	return NewFromString(content)
}

func (l *FromString) String() string {
	return fmt.Sprintf("loader.FromString{Chars: %d}", len(l.content))
}

func (l *FromString) GetReader() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(l.content)), nil
}

// GetSourceURL returns the source URL of the script.
func (l *FromString) GetSourceURL() *url.URL {
	return l.sourceURL
}
