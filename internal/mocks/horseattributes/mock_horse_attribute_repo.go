package mockhorseattributes

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/internal/horseattributes/models"
	"github.com/stretchr/testify/mock"
)

type MockHorseAttributeRepository struct {
	mock.Mock
}

func (m *MockHorseAttributeRepository) Create(ctx context.Context, option *models.HorseAttributeOption) (*models.HorseAttributeOption, error) {
	args := m.Called(ctx, option)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.HorseAttributeOption), args.Error(1)
}

func (m *MockHorseAttributeRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockHorseAttributeRepository) FindAll(ctx context.Context, attrType string) ([]*models.HorseAttributeOption, error) {
	args := m.Called(ctx, attrType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.HorseAttributeOption), args.Error(1)
}
