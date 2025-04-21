package loader

import (
	"io"
	"net/url"
)

// Loader is an interface used by the engines to load scripts or binaries.
type Loader interface {
	GetReader() (io.ReadCloser, error)
	GetSourceURL() *url.URL
}
