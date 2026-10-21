package mockboosttiers

import (
	"context"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/internal/boosttiers/models"
	"github.com/stretchr/testify/mock"
)

type MockBoostTierRepository struct {
	mock.Mock
}

func (m *MockBoostTierRepository) ListAll(ctx context.Context) ([]models.BoostTier, error) {
	args := m.Called(ctx)
	tiers, _ := args.Get(0).([]models.BoostTier)
	return tiers, args.Error(1)
}

func (m *MockBoostTierRepository) ListActive(ctx context.Context) ([]models.BoostTier, error) {
	args := m.Called(ctx)
	tiers, _ := args.Get(0).([]models.BoostTier)
	return tiers, args.Error(1)
}

func (m *MockBoostTierRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.BoostTier, error) {
	args := m.Called(ctx, id)
	tier, _ := args.Get(0).(*models.BoostTier)
	return tier, args.Error(1)
}

func (m *MockBoostTierRepository) FindActiveByCode(ctx context.Context, code string) (*models.BoostTier, error) {
	args := m.Called(ctx, code)
	tier, _ := args.Get(0).(*models.BoostTier)
	return tier, args.Error(1)
}

func (m *MockBoostTierRepository) Create(ctx context.Context, tier *models.BoostTier) (bool, error) {
	args := m.Called(ctx, tier)
	return args.Bool(0), args.Error(1)
}

func (m *MockBoostTierRepository) Update(ctx context.Context, tier *models.BoostTier) (bool, error) {
	args := m.Called(ctx, tier)
	return args.Bool(0), args.Error(1)
}
