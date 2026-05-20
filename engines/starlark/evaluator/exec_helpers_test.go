package evaluator

import (
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// loaderMock satisfies script/loader.Loader for tests that don't care about
// the loader's behavior, only that NewExecutableUnit can call GetReader().
type loaderMock struct{ mock.Mock }

func (m *loaderMock) GetReader() (io.ReadCloser, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *loaderMock) GetSourceURL() *url.URL {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*url.URL)
}

// compilerMock satisfies script.Compiler and returns a preconfigured
// ExecutableContent (which may be nil).
type compilerMock struct{ mock.Mock }

func (m *compilerMock) Compile(r io.ReadCloser) (script.ExecutableContent, error) {
	args := m.Called(r)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(script.ExecutableContent), args.Error(1)
}

func (m *compilerMock) String() string { return "compilerMock" }

// newExe builds an ExecutableUnit by routing through script.NewExecutableUnit
// with a mock loader+compiler that produce the supplied content. Use a
// non-empty id when content is nil, since NewExecutableUnit otherwise tries
// to derive the id from content.GetSource() and panics on nil.
func newExe(
	t *testing.T,
	id string,
	content script.ExecutableContent,
	provider data.Provider,
) *script.ExecutableUnit {
	t.Helper()
	if id == "" {
		id = t.Name()
	}
	ldr := new(loaderMock)
	ldr.On("GetReader").
		Return(io.NopCloser(strings.NewReader("dummy")), nil)
	cmp := new(compilerMock)
	cmp.On("Compile", mock.Anything).Return(content, nil)
	exe, err := script.NewExecutableUnit(nil, id, ldr, cmp, provider)
	require.NoError(t, err)
	return exe
}
