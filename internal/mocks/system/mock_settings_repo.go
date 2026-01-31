package system

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockSettingsRepo struct {
	mock.Mock
}

func (m *MockSettingsRepo) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockSettingsRepo) Set(ctx context.Context, key string, value string) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockSettingsRepo) IsProductApprovalRequired(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}
