package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	categoriesmodels "github.com/hfleury/horsemarketplacebk/internal/categories/models"
	media "github.com/hfleury/horsemarketplacebk/internal/media"
	mockconfig "github.com/hfleury/horsemarketplacebk/internal/mocks/config"
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
		Service:       &models.Service{ProductID: productID, ServiceType: &serviceType},
		Property:      &models.Property{ProductID: productID, SizeM2: &sizeM2},
		FavoriteCount: 3,
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
	assert.Equal(t, fullyPopulated.FavoriteCount, result.FavoriteCount)
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

func TestFindSimilar_ReturnsRepoResults(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

	productID := uuid.New()
	source := &models.Product{ID: productID, Type: models.TypeHorse}
	similar := []*models.Product{{Title: "Similar Horse"}}

	mockRepo.On("FindByID", mock.Anything, productID.String()).Return(source, nil)
	mockRepo.On("FindSimilar", mock.Anything, source, 6).Return(similar, nil)

	result, err := service.FindSimilar(context.Background(), productID.String(), 6)

	assert.NoError(t, err)
	assert.Equal(t, similar, result)
	mockRepo.AssertExpectations(t)
}

func TestFindSimilar_SourceNotFound_ReturnsNilWithoutCallingRepo(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

	productID := uuid.New()
	mockRepo.On("FindByID", mock.Anything, productID.String()).Return(nil, nil)

	result, err := service.FindSimilar(context.Background(), productID.String(), 6)

	assert.NoError(t, err)
	assert.Nil(t, result)
	mockRepo.AssertNotCalled(t, "FindSimilar", mock.Anything, mock.Anything, mock.Anything)
}

func TestFindSimilar_FindByIDError_PropagatesWithoutCallingRepo(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

	productID := uuid.New()
	mockRepo.On("FindByID", mock.Anything, productID.String()).Return(nil, errors.New("db error"))

	result, err := service.FindSimilar(context.Background(), productID.String(), 6)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertNotCalled(t, "FindSimilar", mock.Anything, mock.Anything, mock.Anything)
}

func TestFindSimilar_RepoError_Propagates(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	logger := config.NewZerologService()

	service := services.NewProductService(mockRepo, mockSettings, logger, new(mockGeocoding.MockGeocodingClient))

	productID := uuid.New()
	source := &models.Product{ID: productID}

	mockRepo.On("FindByID", mock.Anything, productID.String()).Return(source, nil)
	mockRepo.On("FindSimilar", mock.Anything, source, 6).Return(nil, errors.New("db error"))

	result, err := service.FindSimilar(context.Background(), productID.String(), 6)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

type publishTestFixture struct {
	service  *services.ProductServiceImp
	repo     *mockProducts.MockProductRepo
	settings *mockSystem.MockSettingsRepo
	notifier *mockProducts.MockListingPublishedNotifier
}

func newPublishTestFixture(logger config.Logging) publishTestFixture {
	fixture := publishTestFixture{
		repo:     new(mockProducts.MockProductRepo),
		settings: new(mockSystem.MockSettingsRepo),
		notifier: new(mockProducts.MockListingPublishedNotifier),
	}
	fixture.service = services.NewProductService(fixture.repo, fixture.settings, logger, new(mockGeocoding.MockGeocodingClient))
	fixture.service.SetPublishNotifier(fixture.notifier)
	return fixture
}

// expectCreateEchoesProduct mirrors the real repository, which returns the
// product it was given after saving it.
func (f publishTestFixture) expectCreateEchoesProduct(product *models.Product) {
	f.repo.On("Create", mock.Anything, product).Return(product, nil)
}

func (f publishTestFixture) expectExistingProduct(product *models.Product) {
	f.repo.On("FindByID", mock.Anything, product.ID.String()).Return(product, nil)
}

func TestCreateProduct_PublishedWithoutApproval_NotifiesOnce(t *testing.T) {
	fixture := newPublishTestFixture(config.NewZerologService())
	fixture.settings.On("IsProductApprovalRequired", mock.Anything).Return(false, nil)
	product := &models.Product{ID: uuid.New(), Status: models.StatusPublished}
	fixture.expectCreateEchoesProduct(product)
	fixture.notifier.On("NotifyListingPublished", mock.Anything, mock.Anything).Return(nil)

	created, err := fixture.service.Create(context.Background(), product)

	assert.NoError(t, err)
	fixture.notifier.AssertCalled(t, "NotifyListingPublished", mock.Anything, created.ID.String())
	fixture.notifier.AssertNumberOfCalls(t, "NotifyListingPublished", 1)
}

func TestCreateProduct_NotPublished_DoesNotNotify(t *testing.T) {
	tests := []struct {
		name             string
		requestedStatus  models.ProductStatus
		approvalRequired bool
		approvalErr      error
	}{
		{name: "approval required makes it pending", requestedStatus: models.StatusPublished, approvalRequired: true},
		{name: "draft request", requestedStatus: models.StatusDraft},
		{name: "approval setting unreadable makes it pending", requestedStatus: models.StatusPublished, approvalErr: errors.New("db error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newPublishTestFixture(config.NewZerologService())
			fixture.settings.On("IsProductApprovalRequired", mock.Anything).Return(tt.approvalRequired, tt.approvalErr)
			product := &models.Product{ID: uuid.New(), Status: tt.requestedStatus}
			fixture.expectCreateEchoesProduct(product)

			created, err := fixture.service.Create(context.Background(), product)

			assert.NoError(t, err)
			assert.NotEqual(t, models.StatusPublished, created.Status)
			fixture.notifier.AssertNotCalled(t, "NotifyListingPublished", mock.Anything, mock.Anything)
		})
	}
}

func TestCreateProduct_RepoError_ReturnsErrorWithoutNotifying(t *testing.T) {
	fixture := newPublishTestFixture(config.NewZerologService())
	fixture.settings.On("IsProductApprovalRequired", mock.Anything).Return(false, nil)
	fixture.repo.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	created, err := fixture.service.Create(context.Background(), &models.Product{Status: models.StatusPublished})

	assert.Error(t, err)
	assert.Nil(t, created)
	fixture.notifier.AssertNotCalled(t, "NotifyListingPublished", mock.Anything, mock.Anything)
}

func TestCreateProduct_WithoutNotifier_PublishesNormally(t *testing.T) {
	mockRepo := new(mockProducts.MockProductRepo)
	mockSettings := new(mockSystem.MockSettingsRepo)
	service := services.NewProductService(mockRepo, mockSettings, config.NewZerologService(), new(mockGeocoding.MockGeocodingClient))
	mockSettings.On("IsProductApprovalRequired", mock.Anything).Return(false, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(&models.Product{Status: models.StatusPublished}, nil)

	created, err := service.Create(context.Background(), &models.Product{Status: models.StatusPublished})

	assert.NoError(t, err)
	assert.Equal(t, models.StatusPublished, created.Status)
}

func TestCreateProduct_NotifierError_StillSucceedsAndLogsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := mockconfig.NewMockLogging(ctrl)
	logger.EXPECT().Log(gomock.Any(), config.ErrorLevel, "Failed to notify that a listing was published", gomock.Any()).Times(1)
	fixture := newPublishTestFixture(logger)
	fixture.settings.On("IsProductApprovalRequired", mock.Anything).Return(false, nil)
	product := &models.Product{ID: uuid.New(), Status: models.StatusPublished}
	fixture.expectCreateEchoesProduct(product)
	fixture.notifier.On("NotifyListingPublished", mock.Anything, mock.Anything).Return(errors.New("redis down"))

	created, err := fixture.service.Create(context.Background(), product)

	assert.NoError(t, err)
	assert.Equal(t, models.StatusPublished, created.Status)
}

func TestUpdateStatus_TransitionIntoPublished_Notifies(t *testing.T) {
	tests := []struct {
		name           string
		previousStatus models.ProductStatus
		isAdmin        bool
	}{
		{name: "admin approves a pending listing", previousStatus: models.StatusPendingApproval, isAdmin: true},
		{name: "owner publishes a draft with approval off", previousStatus: models.StatusDraft},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newPublishTestFixture(config.NewZerologService())
			product := &models.Product{ID: uuid.New(), UserID: uuid.New(), Status: tt.previousStatus}
			fixture.expectExistingProduct(product)
			fixture.settings.On("IsProductApprovalRequired", mock.Anything).Return(false, nil)
			fixture.repo.On("UpdateStatus", mock.Anything, product.ID.String(), models.StatusPublished).Return(nil)
			fixture.notifier.On("NotifyListingPublished", mock.Anything, product.ID.String()).Return(nil)

			err := fixture.service.UpdateStatus(context.Background(), product.ID.String(), models.StatusPublished, product.UserID.String(), tt.isAdmin)

			assert.NoError(t, err)
			fixture.notifier.AssertNumberOfCalls(t, "NotifyListingPublished", 1)
		})
	}
}

func TestUpdateStatus_NoTransitionIntoPublished_DoesNotNotify(t *testing.T) {
	tests := []struct {
		name             string
		previousStatus   models.ProductStatus
		requestedStatus  models.ProductStatus
		savedStatus      models.ProductStatus
		isAdmin          bool
		approvalRequired bool
	}{
		{name: "admin republishes a published listing", previousStatus: models.StatusPublished, requestedStatus: models.StatusPublished, savedStatus: models.StatusPublished, isAdmin: true},
		{name: "owner publish with approval on goes to pending", previousStatus: models.StatusDraft, requestedStatus: models.StatusPublished, savedStatus: models.StatusPendingApproval, approvalRequired: true},
		{name: "owner marks a published listing sold", previousStatus: models.StatusPublished, requestedStatus: models.StatusSold, savedStatus: models.StatusSold},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newPublishTestFixture(config.NewZerologService())
			product := &models.Product{ID: uuid.New(), UserID: uuid.New(), Status: tt.previousStatus}
			fixture.expectExistingProduct(product)
			fixture.settings.On("IsProductApprovalRequired", mock.Anything).Return(tt.approvalRequired, nil)
			fixture.repo.On("UpdateStatus", mock.Anything, product.ID.String(), tt.savedStatus).Return(nil)

			err := fixture.service.UpdateStatus(context.Background(), product.ID.String(), tt.requestedStatus, product.UserID.String(), tt.isAdmin)

			assert.NoError(t, err)
			fixture.repo.AssertExpectations(t)
			fixture.notifier.AssertNotCalled(t, "NotifyListingPublished", mock.Anything, mock.Anything)
		})
	}
}

func TestUpdateStatus_RepoError_DoesNotNotify(t *testing.T) {
	fixture := newPublishTestFixture(config.NewZerologService())
	product := &models.Product{ID: uuid.New(), UserID: uuid.New(), Status: models.StatusPendingApproval}
	fixture.expectExistingProduct(product)
	fixture.repo.On("UpdateStatus", mock.Anything, product.ID.String(), models.StatusPublished).Return(errors.New("db error"))

	err := fixture.service.UpdateStatus(context.Background(), product.ID.String(), models.StatusPublished, "", true)

	assert.Error(t, err)
	fixture.notifier.AssertNotCalled(t, "NotifyListingPublished", mock.Anything, mock.Anything)
}

func TestUpdateStatus_Unauthorized_DoesNotNotify(t *testing.T) {
	fixture := newPublishTestFixture(config.NewZerologService())
	product := &models.Product{ID: uuid.New(), UserID: uuid.New(), Status: models.StatusDraft}
	fixture.expectExistingProduct(product)

	err := fixture.service.UpdateStatus(context.Background(), product.ID.String(), models.StatusPublished, uuid.New().String(), false)

	assert.Equal(t, services.ErrUnauthorized, err)
	fixture.notifier.AssertNotCalled(t, "NotifyListingPublished", mock.Anything, mock.Anything)
}
