package loader

import (
	"io"
	"net/url"
)

type Loader interface {
	GetReader() (io.ReadCloser, error)
	GetSourceURL() *url.URL
}
