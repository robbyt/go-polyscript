package polyscript_test

import (
	"context"

	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script"
	"github.com/stretchr/testify/mock"
)

type mockEvaluator struct {
	mock.Mock
}

func (m *mockEvaluator) Eval(ctx context.Context) (platform.EvaluatorResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(platform.EvaluatorResponse), args.Error(1)
}

func (m *mockEvaluator) Reload(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *mockEvaluator) Load(newVersion script.ExecutableUnit) error {
	args := m.Called(newVersion)
	return args.Error(0)
}

func (m *mockEvaluator) AddDataToContext(
	ctx context.Context,
	d ...map[string]any,
) (context.Context, error) {
	args := m.Called(ctx, d)
	return args.Get(0).(context.Context), args.Error(1)
}

type mockEvaluatorResponse struct {
	mock.Mock
}

func (m *mockEvaluatorResponse) Type() data.Types {
	args := m.Called()
	val := args.Get(0)
	if t, ok := val.(data.Types); ok {
		return t
	}
	switch val.(type) {
	case bool:
		return data.BOOL
	case int:
		return data.INT
	case map[string]any, map[any]any:
		return data.MAP
	case string:
		return data.STRING
	default:
		panic("unknown type")
	}
}

func (m *mockEvaluatorResponse) Inspect() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockEvaluatorResponse) Interface() any {
	args := m.Called()
	return args.Get(0)
}

func (m *mockEvaluatorResponse) GetScriptExeID() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockEvaluatorResponse) GetExecTime() string {
	args := m.Called()
	return args.String(0)
}
