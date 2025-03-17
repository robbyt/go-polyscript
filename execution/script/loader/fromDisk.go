package loader

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/robbyt/go-polyscript/internal/helpers"
)

type FromDisk struct {
	path      string
	sourceURL *url.URL
}

func NewFromDisk(path string) (*FromDisk, error) {
	path = strings.TrimPrefix(path, "file://")

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return nil, fmt.Errorf("%w: %s", ErrSchemeUnsupported, path)
	}

	// Reject relative paths
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("%w: relative paths are not supported", ErrScriptNotAvailable)
	}

	path = filepath.Clean(path)

	if path == "" || path == "." || path == "/" || path == "\\" || path == "../" {
		return nil, fmt.Errorf("%w: path is empty or invalid", ErrScriptNotAvailable)
	}

	// Check if the path contains a scheme
	if !strings.Contains(path, "://") {
		path = "file://" + path
	}

	// Prepend "file://" to the path to create a valid URL
	url, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("unable to parse URL: %w", err)
	}

	if url.Scheme != "file" {
		return nil, fmt.Errorf("%w: %s", ErrSchemeUnsupported, path)
	}

	return &FromDisk{
		path:      path,
		sourceURL: url,
	}, nil
}

func (l *FromDisk) String() string {
	var chksum string
	noChkSum := fmt.Sprintf("loader.FromDisk{Path: %s}", l.path)

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

	return fmt.Sprintf("loader.FromDisk{Path: %s, SHA256: %s}", l.path, chksum)
}

func (l *FromDisk) GetReader() (io.ReadCloser, error) {
	// Just return a reader for the file
	return os.Open(l.sourceURL.Path)
}

// GetSourceURL returns the source URL of the script.
func (l *FromDisk) GetSourceURL() *url.URL {
	return l.sourceURL
}
