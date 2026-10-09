package mocksavedsearches

import (
	"context"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/internal/savedsearches/models"
	"github.com/stretchr/testify/mock"
)

type MockSavedSearchRepository struct {
	mock.Mock
}

func (m *MockSavedSearchRepository) Create(ctx context.Context, savedSearch *models.SavedSearch) error {
	args := m.Called(ctx, savedSearch)
	return args.Error(0)
}

func (m *MockSavedSearchRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]models.SavedSearch, error) {
	args := m.Called(ctx, userID)
	savedSearches, _ := args.Get(0).([]models.SavedSearch)
	return savedSearches, args.Error(1)
}

func (m *MockSavedSearchRepository) FindByIDForUser(ctx context.Context, id, userID uuid.UUID) (*models.SavedSearch, error) {
	args := m.Called(ctx, id, userID)
	savedSearch, _ := args.Get(0).(*models.SavedSearch)
	return savedSearch, args.Error(1)
}

func (m *MockSavedSearchRepository) Update(ctx context.Context, savedSearch *models.SavedSearch) (*models.SavedSearch, error) {
	args := m.Called(ctx, savedSearch)
	updated, _ := args.Get(0).(*models.SavedSearch)
	return updated, args.Error(1)
}

func (m *MockSavedSearchRepository) Delete(ctx context.Context, id, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, id, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockSavedSearchRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}
