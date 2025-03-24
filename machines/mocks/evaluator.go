package mocks

import (
	"context"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/stretchr/testify/mock"
)

// Evaluator is a mock implementation of engine.EvaluatorWithPrep for testing purposes.
type Evaluator struct {
	mock.Mock
}

// Eval is a mock implementation of the Eval method.
func (m *Evaluator) Eval(ctx context.Context) (engine.EvaluatorResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(engine.EvaluatorResponse), args.Error(1)
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

// PrepareContext is a mock implementation of the PrepareContext method.
func (m *Evaluator) PrepareContext(ctx context.Context, d ...any) (context.Context, error) {
	args := m.Called(ctx, d)
	return args.Get(0).(context.Context), args.Error(1)
}
