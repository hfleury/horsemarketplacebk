package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/favorites/services"
	mockfavorites "github.com/hfleury/horsemarketplacebk/internal/mocks/favorites"
	mockProducts "github.com/hfleury/horsemarketplacebk/internal/mocks/products"
	productModels "github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newFavoriteService() (*services.FavoriteService, *mockfavorites.MockFavoriteRepository, *mockProducts.MockProductRepo) {
	mockFavoriteRepo := new(mockfavorites.MockFavoriteRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewFavoriteService(mockFavoriteRepo, mockProductRepo, logger)
	return service, mockFavoriteRepo, mockProductRepo
}

func publishedProduct(productID, sellerID uuid.UUID) *productModels.Product {
	return &productModels.Product{ID: productID, UserID: sellerID, Status: productModels.StatusPublished}
}

func TestFavoriteService_Add_Success(t *testing.T) {
	service, mockFavoriteRepo, mockProductRepo := newFavoriteService()
	productID, sellerID, buyerID := uuid.New(), uuid.New(), uuid.New()

	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(publishedProduct(productID, sellerID), nil)
	mockFavoriteRepo.On("Add", mock.Anything, buyerID, productID).Return(nil)

	status, err := service.Add(context.Background(), buyerID.String(), productID.String())

	assert.NoError(t, err)
	assert.Equal(t, productID, status.ProductID)
	assert.True(t, status.Favorited)
	mockProductRepo.AssertExpectations(t)
	mockFavoriteRepo.AssertExpectations(t)
}

func TestFavoriteService_Add_InvalidProductID(t *testing.T) {
	service, mockFavoriteRepo, mockProductRepo := newFavoriteService()

	status, err := service.Add(context.Background(), uuid.New().String(), "not-a-uuid")

	assert.Nil(t, status)
	assert.ErrorIs(t, err, services.ErrInvalidProductID)
	mockProductRepo.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
	mockFavoriteRepo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
}

func TestFavoriteService_Add_ProductNotFound(t *testing.T) {
	service, mockFavoriteRepo, mockProductRepo := newFavoriteService()
	productID := uuid.New()

	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(nil, nil)

	status, err := service.Add(context.Background(), uuid.New().String(), productID.String())

	assert.Nil(t, status)
	assert.ErrorIs(t, err, services.ErrProductNotFound)
	mockFavoriteRepo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
}

func TestFavoriteService_Add_OwnListing(t *testing.T) {
	service, mockFavoriteRepo, mockProductRepo := newFavoriteService()
	productID, sellerID := uuid.New(), uuid.New()

	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(publishedProduct(productID, sellerID), nil)

	status, err := service.Add(context.Background(), sellerID.String(), productID.String())

	assert.Nil(t, status)
	assert.ErrorIs(t, err, services.ErrCannotFavoriteOwnListing)
	mockFavoriteRepo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
}

func TestFavoriteService_Add_NotPublished(t *testing.T) {
	service, mockFavoriteRepo, mockProductRepo := newFavoriteService()
	productID, sellerID := uuid.New(), uuid.New()
	soldProduct := &productModels.Product{ID: productID, UserID: sellerID, Status: productModels.StatusSold}

	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(soldProduct, nil)

	status, err := service.Add(context.Background(), uuid.New().String(), productID.String())

	assert.Nil(t, status)
	assert.ErrorIs(t, err, services.ErrListingNotPublished)
	mockFavoriteRepo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
}

func TestFavoriteService_Add_RepoError(t *testing.T) {
	service, mockFavoriteRepo, mockProductRepo := newFavoriteService()
	productID, sellerID, buyerID := uuid.New(), uuid.New(), uuid.New()
	repoErr := errors.New("db down")

	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(publishedProduct(productID, sellerID), nil)
	mockFavoriteRepo.On("Add", mock.Anything, buyerID, productID).Return(repoErr)

	status, err := service.Add(context.Background(), buyerID.String(), productID.String())

	assert.Nil(t, status)
	assert.ErrorIs(t, err, repoErr)
}

func TestFavoriteService_Remove_Success(t *testing.T) {
	service, mockFavoriteRepo, mockProductRepo := newFavoriteService()
	productID, buyerID := uuid.New(), uuid.New()

	mockFavoriteRepo.On("Remove", mock.Anything, buyerID, productID).Return(nil)

	status, err := service.Remove(context.Background(), buyerID.String(), productID.String())

	assert.NoError(t, err)
	assert.Equal(t, productID, status.ProductID)
	assert.False(t, status.Favorited)
	mockFavoriteRepo.AssertExpectations(t)
	mockProductRepo.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
}

func TestFavoriteService_Remove_InvalidProductID(t *testing.T) {
	service, mockFavoriteRepo, _ := newFavoriteService()

	status, err := service.Remove(context.Background(), uuid.New().String(), "not-a-uuid")

	assert.Nil(t, status)
	assert.ErrorIs(t, err, services.ErrInvalidProductID)
	mockFavoriteRepo.AssertNotCalled(t, "Remove", mock.Anything, mock.Anything, mock.Anything)
}

func TestFavoriteService_List_ReturnsPaginatedProducts(t *testing.T) {
	service, _, mockProductRepo := newFavoriteService()
	userID := uuid.New().String()
	items := []*productModels.Product{{ID: uuid.New()}, {ID: uuid.New()}}

	mockProductRepo.On("FindFavoritedByUserID", mock.Anything, userID, 2, 10).Return(items, 12, nil)

	result, err := service.List(context.Background(), userID, 2, 10)

	assert.NoError(t, err)
	assert.Equal(t, items, result.Items)
	assert.Equal(t, 12, result.Total)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 10, result.Limit)
	mockProductRepo.AssertExpectations(t)
}

func TestFavoriteService_List_RepoError(t *testing.T) {
	service, _, mockProductRepo := newFavoriteService()
	userID := uuid.New().String()
	repoErr := errors.New("db down")

	mockProductRepo.On("FindFavoritedByUserID", mock.Anything, userID, 1, 20).Return(nil, 0, repoErr)

	result, err := service.List(context.Background(), userID, 1, 20)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, repoErr)
}
