package script

import (
	"context"
	"io"
	"sync/atomic"

	engineTypes "github.com/robbyt/go-polyscript/engines/types"
	"github.com/stretchr/testify/mock"
)

// MockCompiler is a mock implementation of the Compiler interface.
type MockCompiler struct {
	mock.Mock
}

// Compile mocks the Compile method of the Compiler interface.
func (m *MockCompiler) Compile(ctx context.Context, scriptReader io.ReadCloser) (ExecutableContent, error) {
	args := m.Called(ctx, scriptReader)
	execContent, ok := args.Get(0).(ExecutableContent)
	if !ok {
		return nil, args.Error(1)
	}
	return execContent, args.Error(1)
}

// MockExecutableContent is a mock implementation of the ExecutableContent interface for testing.
type MockExecutableContent struct {
	mock.Mock
}

func (m *MockExecutableContent) GetSource() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockExecutableContent) GetByteCode() any {
	args := m.Called()
	return args.Get(0)
}

func (m *MockExecutableContent) EngineType() engineTypes.Type {
	args := m.Called()
	return args.Get(0).(engineTypes.Type)
}

// MockClosableExecutableContent is a MockExecutableContent that also
// implements Close, standing in for compiled content that holds resources.
type MockClosableExecutableContent struct {
	MockExecutableContent
}

func (m *MockClosableExecutableContent) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// countingClosableContent is closable content that counts Close calls and
// returns err from each. It does not use testify expectations, so a Close
// that runs from a context.AfterFunc goroutine cannot panic the test.
type countingClosableContent struct {
	MockExecutableContent
	closeCalls atomic.Int32
	err        error
}

func (c *countingClosableContent) Close(context.Context) error {
	c.closeCalls.Add(1)
	return c.err
}

// blockingClosableContent counts Close calls and blocks each one until gate
// is closed.
type blockingClosableContent struct {
	MockExecutableContent
	gate       chan struct{}
	closeCalls atomic.Int32
}

func (c *blockingClosableContent) Close(context.Context) error {
	c.closeCalls.Add(1)
	<-c.gate
	return nil
}
