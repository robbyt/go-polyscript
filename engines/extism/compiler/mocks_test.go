package compiler

import (
	"context"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/engines/extism/adapters"
	"github.com/stretchr/testify/mock"
)

// mockCompiledPlugin implements adapters.CompiledPlugin for testing the Compile()
// branches that depend on plugin.Instance(...) outcomes.
type mockCompiledPlugin struct {
	mock.Mock
}

func (m *mockCompiledPlugin) Instance(
	ctx context.Context,
	config extismSDK.PluginInstanceConfig,
) (adapters.PluginInstance, error) {
	args := m.Called(ctx, config)
	inst, _ := args.Get(0).(adapters.PluginInstance)
	return inst, args.Error(1)
}

func (m *mockCompiledPlugin) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// mockPluginInstance implements adapters.PluginInstance for testing the Compile()
// branches that depend on FunctionExists / Close outcomes.
type mockPluginInstance struct {
	mock.Mock
}

func (m *mockPluginInstance) Call(name string, data []byte) (uint32, []byte, error) {
	args := m.Called(name, data)
	out, _ := args.Get(1).([]byte)
	return uint32(args.Int(0)), out, args.Error(2)
}

func (m *mockPluginInstance) CallWithContext(
	ctx context.Context,
	name string,
	data []byte,
) (uint32, []byte, error) {
	args := m.Called(ctx, name, data)
	out, _ := args.Get(1).([]byte)
	return uint32(args.Int(0)), out, args.Error(2)
}

func (m *mockPluginInstance) FunctionExists(name string) bool {
	args := m.Called(name)
	return args.Bool(0)
}

func (m *mockPluginInstance) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// errReader is an io.ReadCloser whose Read returns a configured error, so tests
// can drive the io.ReadAll failure branch in Compile().
type errReader struct {
	readErr  error
	closeErr error
}

func (r *errReader) Read(p []byte) (int, error) {
	return 0, r.readErr
}

func (r *errReader) Close() error {
	return r.closeErr
}
