package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	categoriesmodels "github.com/hfleury/horsemarketplacebk/internal/categories/models"
	media "github.com/hfleury/horsemarketplacebk/internal/media"
	mockGeocoding "github.com/hfleury/horsemarketplacebk/internal/mocks/geocoding"
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

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

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

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

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

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

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

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

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

func TestFindByID_PassesThroughRepoResult(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

	productID := uuid.New()
	catID := uuid.New()
	catName := "Horses"
	serviceType := "boarding"
	sizeM2 := 120
	fullyPopulated := &models.Product{
		ID:    productID,
		Title: "Fully Populated Product",
		Type:  models.TypeHorse,
		Category: &categoriesmodels.Category{
			Id:   &catID,
			Name: &catName,
		},
		Media: []models.ProductMedia{
			{ProductID: productID, MediaID: uuid.New(), Order: 0, IsPrimary: true, Media: &media.Media{URL: "https://example.com/img.jpg"}},
		},
		Service:  &models.Service{ProductID: productID, ServiceType: &serviceType},
		Property: &models.Property{ProductID: productID, SizeM2: &sizeM2},
	}

	mockRepo.On("FindByID", mock.Anything, productID.String()).Return(fullyPopulated, nil)
	mockRepo.On("IncrementViewCount", mock.Anything, productID.String()).Return(nil)

	result, err := service.FindByID(context.Background(), productID.String())

	assert.NoError(t, err)
	assert.Same(t, fullyPopulated, result)
	assert.Equal(t, fullyPopulated.Category, result.Category)
	assert.Equal(t, fullyPopulated.Media, result.Media)
	assert.Equal(t, fullyPopulated.Service, result.Service)
	assert.Equal(t, fullyPopulated.Property, result.Property)
	mockRepo.AssertExpectations(t)
}

func TestFindByID_IncrementViewCountFailure_DoesNotFailRequest(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

	productID := uuid.New()
	product := &models.Product{ID: productID, Title: "Test Product"}

	mockRepo.On("FindByID", mock.Anything, productID.String()).Return(product, nil)
	mockRepo.On("IncrementViewCount", mock.Anything, productID.String()).Return(errors.New("db error"))

	result, err := service.FindByID(context.Background(), productID.String())

	assert.NoError(t, err)
	assert.Same(t, product, result)
	mockRepo.AssertExpectations(t)
}

func TestFindByID_NotFound_DoesNotIncrementViewCount(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

	productID := uuid.New()

	mockRepo.On("FindByID", mock.Anything, productID.String()).Return(nil, nil)

	result, err := service.FindByID(context.Background(), productID.String())

	assert.NoError(t, err)
	assert.Nil(t, result)
	mockRepo.AssertNotCalled(t, "IncrementViewCount", mock.Anything, mock.Anything)
}

func TestSearch_ByCategory_Paginated(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

	categoryID := uuid.New().String()
	items := []*models.Product{{Title: "Horse 1"}, {Title: "Horse 2"}}

	// categoryID/query-only search now goes through the unified SearchByFilter path.
	mockRepo.On("SearchByFilter", mock.Anything, categoryID, "", (*models.HorseFilter)(nil), (*models.LocationFilter)(nil), 2, 10).Return(items, 25, nil)

	result, err := service.Search(context.Background(), "", categoryID, nil, nil, nil, 2, 10)

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

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

	items := []*models.Product{{Title: "Item 1"}}

	mockRepo.On("SearchByFilter", mock.Anything, "", "", (*models.HorseFilter)(nil), (*models.LocationFilter)(nil), 1, 20).Return(items, 1, nil)

	result, err := service.Search(context.Background(), "", "", nil, nil, nil, 1, 20)

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

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

	items := []*models.Product{{Title: "A"}, {Title: "B"}, {Title: "C"}}

	// Intentional behavior change from the pre-HM-25 unpaginated query-only wrap:
	// query-only search now shares the paginated SearchByFilter path with every
	// other filter combination (see plan Risks/rollback).
	mockRepo.On("SearchByFilter", mock.Anything, "", "saddle", (*models.HorseFilter)(nil), (*models.LocationFilter)(nil), 1, 20).Return(items, 3, nil)

	result, err := service.Search(context.Background(), "saddle", "", nil, nil, nil, 1, 20)

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

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

	items := []*models.Product{{Title: "Truck"}}
	fieldMap := map[string]string{"make": "Volvo"}

	mockRepo.On("FindByField", mock.Anything, "make", "Volvo").Return(items, nil)

	result, err := service.Search(context.Background(), "", "", fieldMap, nil, nil, 1, 20)

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

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

	items := []*models.Product{{Title: "Warmblood"}}
	breed := "Warmblood"
	minPrice := 10000.0
	maxPrice := 50000.0
	horseFilter := &models.HorseFilter{
		Breed:    &breed,
		MinPrice: &minPrice,
		MaxPrice: &maxPrice,
	}

	mockRepo.On("SearchByFilter", mock.Anything, "", "", horseFilter, (*models.LocationFilter)(nil), 1, 20).Return(items, 1, nil)

	result, err := service.Search(context.Background(), "", "", nil, horseFilter, nil, 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, items, result.Items)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.Limit)
	mockRepo.AssertExpectations(t)
}

func TestSearch_ByLocationFilter_ForwardsToRepo(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

	items := []*models.Product{{Title: "Nearby Horse"}}
	locationFilter := &models.LocationFilter{Lat: 59.3293, Lng: 18.0686, RadiusKm: 50}

	mockRepo.On("SearchByFilter", mock.Anything, "", "", (*models.HorseFilter)(nil), locationFilter, 1, 20).Return(items, 1, nil)

	result, err := service.Search(context.Background(), "", "", nil, nil, locationFilter, 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, items, result.Items)
	assert.Equal(t, 1, result.Total)
	mockRepo.AssertExpectations(t)
}
