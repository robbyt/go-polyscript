package extism

import (
	"io"
	"net/url"

	"github.com/stretchr/testify/mock"
)

type loaderMock struct {
	mock.Mock
}

func (m *loaderMock) GetSourceURL() *url.URL {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*url.URL)
}

func (m *loaderMock) GetReader() (io.ReadCloser, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}
