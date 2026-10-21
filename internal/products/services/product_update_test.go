package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/geocoding"
	mockGeocoding "github.com/hfleury/horsemarketplacebk/internal/mocks/geocoding"
	mockProducts "github.com/hfleury/horsemarketplacebk/internal/mocks/products"
	mockSystem "github.com/hfleury/horsemarketplacebk/internal/mocks/system"
	"github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/hfleury/horsemarketplacebk/internal/products/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type updateTestFixture struct {
	service  *services.ProductServiceImp
	repo     *mockProducts.MockProductRepo
	settings *mockSystem.MockSettingsRepo
	geocoder *mockGeocoding.MockGeocodingClient
}

func newUpdateTestFixture() updateTestFixture {
	fixture := updateTestFixture{
		repo:     new(mockProducts.MockProductRepo),
		settings: new(mockSystem.MockSettingsRepo),
		geocoder: new(mockGeocoding.MockGeocodingClient),
	}
	fixture.service = services.NewProductService(fixture.repo, fixture.settings, config.NewZerologService(), fixture.geocoder)
	return fixture
}

func stringPointer(value string) *string { return &value }

func floatPointer(value float64) *float64 { return &value }

func storedHorseListing(status models.ProductStatus) *models.Product {
	categoryID := uuid.New()
	return &models.Product{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		CategoryID:  &categoryID,
		Type:        models.TypeHorse,
		Status:      status,
		Title:       "Old title",
		City:        stringPointer("Uppsala"),
		Area:        stringPointer("Centrum"),
		Latitude:    floatPointer(59.85),
		Longitude:   floatPointer(17.63),
		ViewsCount:  12,
		Horse:       &models.Horse{Name: stringPointer("Old name")},
		Description: stringPointer("Old description"),
	}
}

func sameLocationChanges(existing *models.Product) *models.Product {
	return &models.Product{
		Title:           "New title",
		PriceSEK:        floatPointer(45000),
		Description:     stringPointer("New description"),
		City:            existing.City,
		Area:            existing.Area,
		TransactionType: stringPointer("Rent"),
		Horse:           &models.Horse{Name: stringPointer("New name")},
	}
}

// expectSave expects one Update matching the given predicate and then serves
// the reloaded listing from the second FindByID.
func (f updateTestFixture) expectSave(existing, reloaded *models.Product, matches func(*models.Product) bool) {
	f.repo.On("FindByID", mock.Anything, existing.ID.String()).Return(existing, nil).Once()
	f.repo.On("Update", mock.Anything, mock.MatchedBy(matches)).Return(nil).Once()
	f.repo.On("FindByID", mock.Anything, existing.ID.String()).Return(reloaded, nil).Once()
}

func TestUpdate_OwnerSavesEditableFields_ReturnsReloadedListing(t *testing.T) {
	fixture := newUpdateTestFixture()
	existing := storedHorseListing(models.StatusDraft)
	changes := sameLocationChanges(existing)
	reloaded := &models.Product{ID: existing.ID, Title: "New title"}
	fixture.expectSave(existing, reloaded, func(p *models.Product) bool {
		return p.ID == existing.ID &&
			p.Title == "New title" &&
			*p.PriceSEK == 45000 &&
			*p.Description == "New description" &&
			*p.TransactionType == "Rent" &&
			*p.Horse.Name == "New name"
	})

	result, err := fixture.service.Update(context.Background(), existing.ID.String(), changes, existing.UserID.String(), false)

	assert.NoError(t, err)
	assert.Same(t, reloaded, result)
	fixture.repo.AssertExpectations(t)
}

func TestUpdate_MissingOrDeletedListing_ReturnsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		existing *models.Product
	}{
		{name: "missing", existing: nil},
		{name: "deleted", existing: storedHorseListing(models.StatusDeleted)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newUpdateTestFixture()
			id := uuid.New().String()
			if tt.existing != nil {
				id = tt.existing.ID.String()
				fixture.repo.On("FindByID", mock.Anything, id).Return(tt.existing, nil)
			} else {
				fixture.repo.On("FindByID", mock.Anything, id).Return(nil, nil)
			}

			result, err := fixture.service.Update(context.Background(), id, &models.Product{}, uuid.New().String(), true)

			assert.ErrorIs(t, err, services.ErrProductNotFound)
			assert.Nil(t, result)
			fixture.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		})
	}
}

func TestUpdate_FindByIDError_Propagates(t *testing.T) {
	fixture := newUpdateTestFixture()
	id := uuid.New().String()
	fixture.repo.On("FindByID", mock.Anything, id).Return(nil, errors.New("db error"))

	result, err := fixture.service.Update(context.Background(), id, &models.Product{}, uuid.New().String(), false)

	assert.EqualError(t, err, "db error")
	assert.Nil(t, result)
	fixture.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestUpdate_NonOwnerNonAdmin_ReturnsUnauthorized(t *testing.T) {
	fixture := newUpdateTestFixture()
	existing := storedHorseListing(models.StatusDraft)
	fixture.repo.On("FindByID", mock.Anything, existing.ID.String()).Return(existing, nil)

	result, err := fixture.service.Update(context.Background(), existing.ID.String(), sameLocationChanges(existing), uuid.New().String(), false)

	assert.ErrorIs(t, err, services.ErrUnauthorized)
	assert.Nil(t, result)
	fixture.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestUpdate_AdminNonOwner_Succeeds(t *testing.T) {
	fixture := newUpdateTestFixture()
	existing := storedHorseListing(models.StatusDraft)
	fixture.expectSave(existing, existing, func(p *models.Product) bool { return p.Title == "New title" })

	_, err := fixture.service.Update(context.Background(), existing.ID.String(), sameLocationChanges(existing), uuid.New().String(), true)

	assert.NoError(t, err)
	fixture.repo.AssertExpectations(t)
}

func TestUpdate_StatusAfterEdit(t *testing.T) {
	tests := []struct {
		name             string
		storedStatus     models.ProductStatus
		isAdmin          bool
		approvalRequired bool
		approvalErr      error
		savedStatus      models.ProductStatus
	}{
		{name: "owner edit of published with approval required goes to pending", storedStatus: models.StatusPublished, approvalRequired: true, savedStatus: models.StatusPendingApproval},
		{name: "owner edit of published without approval stays published", storedStatus: models.StatusPublished, savedStatus: models.StatusPublished},
		{name: "unreadable approval setting fails safe to pending", storedStatus: models.StatusPublished, approvalErr: errors.New("db error"), savedStatus: models.StatusPendingApproval},
		{name: "admin edit of published stays published", storedStatus: models.StatusPublished, isAdmin: true, approvalRequired: true, savedStatus: models.StatusPublished},
		{name: "draft stays draft", storedStatus: models.StatusDraft, approvalRequired: true, savedStatus: models.StatusDraft},
		{name: "pending stays pending", storedStatus: models.StatusPendingApproval, approvalRequired: true, savedStatus: models.StatusPendingApproval},
		{name: "sold stays sold", storedStatus: models.StatusSold, approvalRequired: true, savedStatus: models.StatusSold},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newUpdateTestFixture()
			existing := storedHorseListing(tt.storedStatus)
			fixture.settings.On("IsProductApprovalRequired", mock.Anything).Return(tt.approvalRequired, tt.approvalErr)
			fixture.expectSave(existing, existing, func(p *models.Product) bool { return p.Status == tt.savedStatus })

			_, err := fixture.service.Update(context.Background(), existing.ID.String(), sameLocationChanges(existing), existing.UserID.String(), tt.isAdmin)

			assert.NoError(t, err)
			fixture.repo.AssertExpectations(t)
		})
	}
}

func TestUpdate_IgnoresNonEditableFieldsAndForeignSubObjects(t *testing.T) {
	fixture := newUpdateTestFixture()
	existing := storedHorseListing(models.StatusDraft)
	otherCategory := uuid.New()
	changes := sameLocationChanges(existing)
	changes.Type = models.TypeVehicle
	changes.Status = models.StatusPublished
	changes.UserID = uuid.New()
	changes.CategoryID = &otherCategory
	changes.ViewsCount = 999
	changes.Vehicle = &models.Vehicle{Make: stringPointer("Volvo")}
	fixture.expectSave(existing, existing, func(p *models.Product) bool {
		return p.Type == models.TypeHorse &&
			p.Status == models.StatusDraft &&
			p.UserID == existing.UserID &&
			*p.CategoryID == *existing.CategoryID &&
			p.ViewsCount == 12 &&
			p.Vehicle == nil
	})

	_, err := fixture.service.Update(context.Background(), existing.ID.String(), changes, existing.UserID.String(), false)

	assert.NoError(t, err)
	fixture.repo.AssertExpectations(t)
}

func TestUpdate_CityChanged_Regeocodes(t *testing.T) {
	fixture := newUpdateTestFixture()
	existing := storedHorseListing(models.StatusDraft)
	changes := sameLocationChanges(existing)
	changes.City = stringPointer("Malmö")
	changes.Area = nil
	fixture.geocoder.On("Geocode", mock.Anything, "Malmö, Sweden").Return(&geocoding.Coordinates{Lat: 55.6, Lng: 13.0}, nil)
	fixture.expectSave(existing, existing, func(p *models.Product) bool {
		return *p.Latitude == 55.6 && *p.Longitude == 13.0
	})

	_, err := fixture.service.Update(context.Background(), existing.ID.String(), changes, existing.UserID.String(), false)

	assert.NoError(t, err)
	fixture.repo.AssertExpectations(t)
	fixture.geocoder.AssertExpectations(t)
}

func TestUpdate_CityCleared_DropsCoordinatesWithoutGeocoding(t *testing.T) {
	fixture := newUpdateTestFixture()
	existing := storedHorseListing(models.StatusDraft)
	changes := sameLocationChanges(existing)
	changes.City = nil
	changes.Area = nil
	fixture.expectSave(existing, existing, func(p *models.Product) bool {
		return p.Latitude == nil && p.Longitude == nil
	})

	_, err := fixture.service.Update(context.Background(), existing.ID.String(), changes, existing.UserID.String(), false)

	assert.NoError(t, err)
	fixture.geocoder.AssertNotCalled(t, "Geocode", mock.Anything, mock.Anything)
}

func TestUpdate_LocationUnchanged_KeepsCoordinatesWithoutGeocoding(t *testing.T) {
	fixture := newUpdateTestFixture()
	existing := storedHorseListing(models.StatusDraft)
	changes := sameLocationChanges(existing)
	changes.City = stringPointer("Uppsala")
	changes.Area = stringPointer("Centrum")
	fixture.expectSave(existing, existing, func(p *models.Product) bool {
		return *p.Latitude == 59.85 && *p.Longitude == 17.63
	})

	_, err := fixture.service.Update(context.Background(), existing.ID.String(), changes, existing.UserID.String(), false)

	assert.NoError(t, err)
	fixture.geocoder.AssertNotCalled(t, "Geocode", mock.Anything, mock.Anything)
}

func TestUpdate_RepoUpdateError_Propagates(t *testing.T) {
	fixture := newUpdateTestFixture()
	existing := storedHorseListing(models.StatusDraft)
	fixture.repo.On("FindByID", mock.Anything, existing.ID.String()).Return(existing, nil)
	fixture.repo.On("Update", mock.Anything, mock.Anything).Return(errors.New("db error"))

	result, err := fixture.service.Update(context.Background(), existing.ID.String(), sameLocationChanges(existing), existing.UserID.String(), false)

	assert.EqualError(t, err, "db error")
	assert.Nil(t, result)
	fixture.repo.AssertNumberOfCalls(t, "FindByID", 1)
}
