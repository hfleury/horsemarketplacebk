package products

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/stretchr/testify/mock"
)

type MockProductService struct {
	mock.Mock
}

func (m *MockProductService) Create(ctx context.Context, product *models.Product) (*models.Product, error) {
	args := m.Called(ctx, product)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductService) FindByID(ctx context.Context, id string) (*models.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductService) FindAll(ctx context.Context, filters map[string]any, page, limit int) (*models.PaginatedProducts, error) {
	args := m.Called(ctx, filters, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaginatedProducts), args.Error(1)
}

func (m *MockProductService) UpdateStatus(ctx context.Context, id string, status models.ProductStatus, userID string, isAdmin bool) error {
	args := m.Called(ctx, id, status, userID, isAdmin)
	return args.Error(0)
}

func (m *MockProductService) Delete(ctx context.Context, id string, userID string, isAdmin bool) error {
	args := m.Called(ctx, id, userID, isAdmin)
	return args.Error(0)
}

func (m *MockProductService) Search(ctx context.Context, query string, categoryID string, fieldMap map[string]string, horseFilter *models.HorseFilter, locationFilter *models.LocationFilter, page, limit int) (*models.PaginatedProducts, error) {
	args := m.Called(ctx, query, categoryID, fieldMap, horseFilter, locationFilter, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaginatedProducts), args.Error(1)
}
