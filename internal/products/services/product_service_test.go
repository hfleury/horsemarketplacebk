package services_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	mockProducts "github.com/hfleury/horsemarketplacebk/internal/mocks/products"
	mockSystem "github.com/hfleury/horsemarketplacebk/internal/mocks/system"
	"github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/hfleury/horsemarketplacebk/internal/products/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateProduct_ApprovalRequired(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger)

	// Setup: Approval IS required
	mockSettings.On("IsProductApprovalRequired", mock.Anything).Return(true, nil)

	inputProduct := &models.Product{
		Title:  "Test Horse",
		Status: models.StatusPublished, // User tries to publish immediately
		Type:   models.TypeHorse,
	}

	// Expectation: Repo should receive product with StatusPendingApproval
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(p *models.Product) bool {
		return p.Status == models.StatusPendingApproval
	})).Return(inputProduct, nil)

	created, err := service.Create(context.Background(), inputProduct)

	assert.NoError(t, err)
	assert.Equal(t, models.StatusPendingApproval, created.Status)
	mockRepo.AssertExpectations(t)
}

func TestCreateProduct_NoApprovalRequired(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger)

	// Setup: Approval NOT required
	mockSettings.On("IsProductApprovalRequired", mock.Anything).Return(false, nil)

	inputProduct := &models.Product{
		Title:  "Test Horse",
		Status: models.StatusPublished,
	}

	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(p *models.Product) bool {
		return p.Status == models.StatusPublished
	})).Return(inputProduct, nil)

	created, err := service.Create(context.Background(), inputProduct)

	assert.NoError(t, err)
	assert.Equal(t, models.StatusPublished, created.Status)
}

func TestUpdateStatus_UserPublish_ApprovalRequired(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger)

	userID := uuid.New()
	productID := uuid.New()
	existingProduct := &models.Product{
		ID:     productID,
		UserID: userID,
		Status: models.StatusDraft,
	}

	mockRepo.On("FindByID", mock.Anything, productID.String()).Return(existingProduct, nil)
	mockSettings.On("IsProductApprovalRequired", mock.Anything).Return(true, nil)

	// Expect FindByID then UpdateStatus with PENDING
	mockRepo.On("UpdateStatus", mock.Anything, productID.String(), models.StatusPendingApproval).Return(nil)

	err := service.UpdateStatus(context.Background(), productID.String(), models.StatusPublished, userID.String(), false)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateStatus_Unauthorized(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	productID := uuid.New()

	existingProduct := &models.Product{
		ID:     productID,
		UserID: ownerID,
		Status: models.StatusDraft,
	}

	mockRepo.On("FindByID", mock.Anything, productID.String()).Return(existingProduct, nil)

	err := service.UpdateStatus(context.Background(), productID.String(), models.StatusDeleted, otherUserID.String(), false)

	assert.Equal(t, services.ErrUnauthorized, err)
}
