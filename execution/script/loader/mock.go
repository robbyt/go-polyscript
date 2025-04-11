package loader

import (
	"bytes"
	"io"
	"net/url"

	"github.com/stretchr/testify/mock"
)

// MockLoader implements the loader.Loader interface for testing
type MockLoader struct {
	mock.Mock
}

func (m *MockLoader) GetSourceURL() *url.URL {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*url.URL)
}

func (m *MockLoader) GetReader() (io.ReadCloser, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockLoader) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Helper method to easily create a mock with content
func NewMockLoaderWithContent(content []byte) *MockLoader {
	m := new(MockLoader)
	m.On("GetReader").Return(io.NopCloser(bytes.NewReader(content)), nil)
	return m
}
