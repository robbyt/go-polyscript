package loader

import (
	"bytes"
	"fmt"
	"io"
	"net/url"

	"github.com/robbyt/go-polyscript/internal/helpers"
)

// FromBytes implements the Loader interface for content from a byte slice.
type FromBytes struct {
	content   []byte
	sourceURL *url.URL
}

// NewFromBytes creates a new Loader from a byte slice.
func NewFromBytes(content []byte) (*FromBytes, error) {
	if len(content) == 0 {
		return nil, fmt.Errorf("%w: content is empty", ErrScriptNotAvailable)
	}

	// Check if content contains only whitespace, but only if it appears to be text data
	// For binary data with null bytes or high-bit bytes, skip whitespace check
	if !hasBinaryCharacters(content) && isOnlyWhitespace(content) {
		return nil, fmt.Errorf(
			"%w: content is empty or contains only whitespace",
			ErrScriptNotAvailable,
		)
	}

	contentHash := helpers.SHA256Bytes(content)[:8]
	u, err := url.Parse("bytes://inline/" + contentHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create source URL: %w", err)
	}

	return &FromBytes{
		content:   content,
		sourceURL: u,
	}, nil
}

func (l *FromBytes) String() string {
	return fmt.Sprintf("loader.FromBytes{Bytes: %d}", len(l.content))
}

// GetReader returns a new reader for the stored content.
func (l *FromBytes) GetReader() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(l.content)), nil
}

// GetSourceURL returns the source URL of the script.
func (l *FromBytes) GetSourceURL() *url.URL {
	return l.sourceURL
}

// isOnlyWhitespace checks if a byte slice contains only whitespace characters
func isOnlyWhitespace(data []byte) bool {
	if len(data) == 0 {
		return true
	}

	for _, b := range data {
		// Check if byte is not a whitespace character
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' && b != '\f' && b != '\v' {
			return false
		}
	}
	return true
}

// hasBinaryCharacters checks if data contains likely binary (non-text) data
func hasBinaryCharacters(data []byte) bool {
	for _, b := range data {
		// Check for null bytes or control characters (except common whitespace)
		if b == 0 || (b < 32 && b != '\n' && b != '\r' && b != '\t') {
			return true
		}
	}
	return false
}
