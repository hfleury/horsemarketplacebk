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

func TestSearch_ByCategory_Paginated(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger)

	categoryID := uuid.New().String()
	items := []*models.Product{{Title: "Horse 1"}, {Title: "Horse 2"}}

	// categoryID/query-only search now goes through the unified SearchByFilter path.
	mockRepo.On("SearchByFilter", mock.Anything, categoryID, "", (*models.HorseFilter)(nil), 2, 10).Return(items, 25, nil)

	result, err := service.Search(context.Background(), "", categoryID, nil, nil, 2, 10)

	assert.NoError(t, err)
	assert.Equal(t, items, result.Items)
	assert.Equal(t, 25, result.Total)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 10, result.Limit)
	mockRepo.AssertExpectations(t)
}

func TestSearch_NoFilters_Paginated(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger)

	items := []*models.Product{{Title: "Item 1"}}

	mockRepo.On("SearchByFilter", mock.Anything, "", "", (*models.HorseFilter)(nil), 1, 20).Return(items, 1, nil)

	result, err := service.Search(context.Background(), "", "", nil, nil, 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, items, result.Items)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.Limit)
	mockRepo.AssertExpectations(t)
}

func TestSearch_ByQuery_Paginated(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger)

	items := []*models.Product{{Title: "A"}, {Title: "B"}, {Title: "C"}}

	// Intentional behavior change from the pre-HM-25 unpaginated query-only wrap:
	// query-only search now shares the paginated SearchByFilter path with every
	// other filter combination (see plan Risks/rollback).
	mockRepo.On("SearchByFilter", mock.Anything, "", "saddle", (*models.HorseFilter)(nil), 1, 20).Return(items, 3, nil)

	result, err := service.Search(context.Background(), "saddle", "", nil, nil, 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, items, result.Items)
	assert.Equal(t, 3, result.Total)
	assert.Equal(t, 1, result.Page)
	mockRepo.AssertExpectations(t)
}

func TestSearch_ByFieldMap_ReturnsUnpaginatedWrap(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger)

	items := []*models.Product{{Title: "Truck"}}
	fieldMap := map[string]string{"make": "Volvo"}

	mockRepo.On("FindByField", mock.Anything, "make", "Volvo").Return(items, nil)

	result, err := service.Search(context.Background(), "", "", fieldMap, nil, 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, items, result.Items)
	assert.Equal(t, len(items), result.Total)
	assert.Equal(t, 1, result.Page)
	mockRepo.AssertExpectations(t)
}

func TestSearch_ByHorseFilter_CombinesFieldsAndPaginates(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger)

	items := []*models.Product{{Title: "Warmblood"}}
	breed := "Warmblood"
	minPrice := 10000.0
	maxPrice := 50000.0
	horseFilter := &models.HorseFilter{
		Breed:    &breed,
		MinPrice: &minPrice,
		MaxPrice: &maxPrice,
	}

	mockRepo.On("SearchByFilter", mock.Anything, "", "", horseFilter, 1, 20).Return(items, 1, nil)

	result, err := service.Search(context.Background(), "", "", nil, horseFilter, 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, items, result.Items)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.Limit)
	mockRepo.AssertExpectations(t)
}
