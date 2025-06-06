package mocks

import (
	"context"

	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/script"
	"github.com/stretchr/testify/mock"
)

// Evaluator is a mock implementation of evaluation.Evaluator for testing purposes.
type Evaluator struct {
	mock.Mock
}

// Eval is a mock implementation of the Eval method.
func (m *Evaluator) Eval(ctx context.Context) (platform.EvaluatorResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(platform.EvaluatorResponse), args.Error(1)
}

// Reload is a mock implementation of the Reload method.
func (m *Evaluator) Reload(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

// Load is a mock implementation of the Load method.
func (m *Evaluator) Load(newVersion script.ExecutableUnit) error {
	args := m.Called(newVersion)
	return args.Error(0)
}

// AddDataToContext is a mock implementation of the AddDataToContext method.
func (m *Evaluator) AddDataToContext(
	ctx context.Context,
	d ...map[string]any,
) (context.Context, error) {
	args := m.Called(ctx, d)
	return args.Get(0).(context.Context), args.Error(1)
}
