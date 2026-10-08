package mockfavorites

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockFavoriteRepository struct {
	mock.Mock
}

func (m *MockFavoriteRepository) Add(ctx context.Context, userID, productID uuid.UUID) error {
	args := m.Called(ctx, userID, productID)
	return args.Error(0)
}

func (m *MockFavoriteRepository) Remove(ctx context.Context, userID, productID uuid.UUID) error {
	args := m.Called(ctx, userID, productID)
	return args.Error(0)
}

func (m *MockFavoriteRepository) ListProductIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, userID)
	productIDs, _ := args.Get(0).([]uuid.UUID)
	return productIDs, args.Error(1)
}
