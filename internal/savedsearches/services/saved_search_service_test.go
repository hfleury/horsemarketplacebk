package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	mocksavedsearches "github.com/hfleury/horsemarketplacebk/internal/mocks/savedsearches"
	"github.com/hfleury/horsemarketplacebk/internal/savedsearches/models"
	"github.com/hfleury/horsemarketplacebk/internal/savedsearches/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newSavedSearchService() (*services.SavedSearchService, *mocksavedsearches.MockSavedSearchRepository) {
	mockRepo := new(mocksavedsearches.MockSavedSearchRepository)
	logger := config.NewZerologService()
	return services.NewSavedSearchService(mockRepo, logger), mockRepo
}

func stringPtr(value string) *string  { return &value }
func intPtr(value int) *int           { return &value }
func floatPtr(value float64) *float64 { return &value }
func boolPtr(value bool) *bool        { return &value }

func validRequest() models.SavedSearchRequest {
	return models.SavedSearchRequest{
		Name:     "Arabians",
		Criteria: &models.SearchCriteria{Breed: stringPtr("Arabian")},
	}
}

func TestSavedSearchService_Create_Success(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	userID := uuid.New()
	req := models.SavedSearchRequest{
		Name: "  Arabians near Stockholm  ",
		Criteria: &models.SearchCriteria{
			Breed:    stringPtr(" Arabian "),
			Gender:   stringPtr("   "),
			MaxPrice: floatPtr(50000),
			Lat:      floatPtr(59.33),
			Lng:      floatPtr(18.07),
			RadiusKm: floatPtr(50),
		},
		LocationLabel: stringPtr(" Stockholm "),
	}

	mockRepo.On("CountByUserID", mock.Anything, userID).Return(3, nil)
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(s *models.SavedSearch) bool {
		return s.UserID == userID && s.ID != uuid.Nil
	})).Return(nil)

	savedSearch, err := service.Create(context.Background(), userID.String(), req)

	assert.NoError(t, err)
	assert.Equal(t, "Arabians near Stockholm", savedSearch.Name)
	assert.Equal(t, "Arabian", *savedSearch.Criteria.Breed)
	assert.Nil(t, savedSearch.Criteria.Gender)
	assert.Equal(t, "Stockholm", *savedSearch.LocationLabel)
	assert.True(t, savedSearch.AlertsEnabled)
	mockRepo.AssertExpectations(t)
}

func TestSavedSearchService_Create_AlertsDisabled(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	userID := uuid.New()
	req := validRequest()
	req.AlertsEnabled = boolPtr(false)

	mockRepo.On("CountByUserID", mock.Anything, userID).Return(0, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	savedSearch, err := service.Create(context.Background(), userID.String(), req)

	assert.NoError(t, err)
	assert.False(t, savedSearch.AlertsEnabled)
}

func TestSavedSearchService_Create_RejectsInvalidName(t *testing.T) {
	for _, name := range []string{"", "   ", strings.Repeat("a", 101)} {
		service, mockRepo := newSavedSearchService()
		req := validRequest()
		req.Name = name

		savedSearch, err := service.Create(context.Background(), uuid.New().String(), req)

		assert.Nil(t, savedSearch)
		assert.ErrorIs(t, err, services.ErrInvalidName)
		mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	}
}

func TestSavedSearchService_Create_RejectsInvalidCriteria(t *testing.T) {
	cases := map[string]models.SavedSearchRequest{
		"empty criteria":        {Name: "x", Criteria: &models.SearchCriteria{}},
		"only blank strings":    {Name: "x", Criteria: &models.SearchCriteria{Query: stringPtr("  "), Breed: stringPtr("")}},
		"invalid category id":   {Name: "x", Criteria: &models.SearchCriteria{CategoryID: stringPtr("not-a-uuid")}},
		"negative price":        {Name: "x", Criteria: &models.SearchCriteria{MinPrice: floatPtr(-1)}},
		"min age above max":     {Name: "x", Criteria: &models.SearchCriteria{MinAge: intPtr(10), MaxAge: intPtr(5)}},
		"min height above max":  {Name: "x", Criteria: &models.SearchCriteria{MinHeight: intPtr(170), MaxHeight: intPtr(150)}},
		"min price above max":   {Name: "x", Criteria: &models.SearchCriteria{MinPrice: floatPtr(900), MaxPrice: floatPtr(100)}},
		"partial location":      {Name: "x", Criteria: &models.SearchCriteria{Lat: floatPtr(59), Lng: floatPtr(18)}},
		"radius below minimum":  {Name: "x", Criteria: &models.SearchCriteria{Lat: floatPtr(59), Lng: floatPtr(18), RadiusKm: floatPtr(5)}},
		"radius above maximum":  {Name: "x", Criteria: &models.SearchCriteria{Lat: floatPtr(59), Lng: floatPtr(18), RadiusKm: floatPtr(600)}},
		"latitude out of range": {Name: "x", Criteria: &models.SearchCriteria{Lat: floatPtr(91), Lng: floatPtr(18), RadiusKm: floatPtr(50)}},
		"label without location": {
			Name:          "x",
			Criteria:      &models.SearchCriteria{Breed: stringPtr("Arabian")},
			LocationLabel: stringPtr("Uppsala"),
		},
	}

	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			service, mockRepo := newSavedSearchService()

			savedSearch, err := service.Create(context.Background(), uuid.New().String(), req)

			assert.Nil(t, savedSearch)
			assert.ErrorIs(t, err, services.ErrInvalidCriteria)
			mockRepo.AssertNotCalled(t, "CountByUserID", mock.Anything, mock.Anything)
			mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}
}

func TestSavedSearchService_Create_LimitReached(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	userID := uuid.New()

	mockRepo.On("CountByUserID", mock.Anything, userID).Return(20, nil)

	savedSearch, err := service.Create(context.Background(), userID.String(), validRequest())

	assert.Nil(t, savedSearch)
	assert.ErrorIs(t, err, services.ErrSavedSearchLimitReached)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestSavedSearchService_Create_BelowLimit(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	userID := uuid.New()

	mockRepo.On("CountByUserID", mock.Anything, userID).Return(19, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	_, err := service.Create(context.Background(), userID.String(), validRequest())

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSavedSearchService_Create_RepoError(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	userID := uuid.New()
	dbErr := errors.New("db down")

	mockRepo.On("CountByUserID", mock.Anything, userID).Return(0, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := service.Create(context.Background(), userID.String(), validRequest())

	assert.ErrorIs(t, err, dbErr)
}

func TestSavedSearchService_List(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	userID := uuid.New()
	expected := []models.SavedSearch{{ID: uuid.New(), UserID: userID}}

	mockRepo.On("ListByUserID", mock.Anything, userID).Return(expected, nil)

	savedSearches, err := service.List(context.Background(), userID.String())

	assert.NoError(t, err)
	assert.Equal(t, expected, savedSearches)
}

func TestSavedSearchService_Get_Success(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	userID, id := uuid.New(), uuid.New()
	expected := &models.SavedSearch{ID: id, UserID: userID}

	mockRepo.On("FindByIDForUser", mock.Anything, id, userID).Return(expected, nil)

	savedSearch, err := service.Get(context.Background(), userID.String(), id.String())

	assert.NoError(t, err)
	assert.Equal(t, expected, savedSearch)
}

func TestSavedSearchService_Get_InvalidID(t *testing.T) {
	service, mockRepo := newSavedSearchService()

	_, err := service.Get(context.Background(), uuid.New().String(), "not-a-uuid")

	assert.ErrorIs(t, err, services.ErrInvalidSavedSearchID)
	mockRepo.AssertNotCalled(t, "FindByIDForUser", mock.Anything, mock.Anything, mock.Anything)
}

// The repo scopes by user_id, so another user's search also comes back nil.
func TestSavedSearchService_Get_NotFound(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	userID, id := uuid.New(), uuid.New()

	mockRepo.On("FindByIDForUser", mock.Anything, id, userID).Return(nil, nil)

	_, err := service.Get(context.Background(), userID.String(), id.String())

	assert.ErrorIs(t, err, services.ErrSavedSearchNotFound)
}

func TestSavedSearchService_Update_Success(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	userID, id := uuid.New(), uuid.New()
	req := validRequest()
	req.AlertsEnabled = boolPtr(false)

	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(s *models.SavedSearch) bool {
		return s.ID == id && s.UserID == userID && !s.AlertsEnabled
	})).Return(&models.SavedSearch{ID: id, UserID: userID}, nil)

	savedSearch, err := service.Update(context.Background(), userID.String(), id.String(), req)

	assert.NoError(t, err)
	assert.Equal(t, id, savedSearch.ID)
	mockRepo.AssertExpectations(t)
}

func TestSavedSearchService_Update_OmittedAlertsDefaultsToEnabled(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	userID, id := uuid.New(), uuid.New()

	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(s *models.SavedSearch) bool {
		return s.AlertsEnabled
	})).Return(&models.SavedSearch{ID: id}, nil)

	_, err := service.Update(context.Background(), userID.String(), id.String(), validRequest())

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSavedSearchService_Update_NotFound(t *testing.T) {
	service, mockRepo := newSavedSearchService()

	mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil, nil)

	_, err := service.Update(context.Background(), uuid.New().String(), uuid.New().String(), validRequest())

	assert.ErrorIs(t, err, services.ErrSavedSearchNotFound)
}

func TestSavedSearchService_Update_InvalidCriteria(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	req := validRequest()
	req.Criteria = &models.SearchCriteria{}

	_, err := service.Update(context.Background(), uuid.New().String(), uuid.New().String(), req)

	assert.ErrorIs(t, err, services.ErrInvalidCriteria)
	mockRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestSavedSearchService_Delete_Success(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	userID, id := uuid.New(), uuid.New()

	mockRepo.On("Delete", mock.Anything, id, userID).Return(true, nil)

	err := service.Delete(context.Background(), userID.String(), id.String())

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSavedSearchService_Delete_NotFound(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	userID, id := uuid.New(), uuid.New()

	mockRepo.On("Delete", mock.Anything, id, userID).Return(false, nil)

	err := service.Delete(context.Background(), userID.String(), id.String())

	assert.ErrorIs(t, err, services.ErrSavedSearchNotFound)
}

func TestSavedSearchService_Delete_InvalidID(t *testing.T) {
	service, mockRepo := newSavedSearchService()

	err := service.Delete(context.Background(), uuid.New().String(), "bad")

	assert.ErrorIs(t, err, services.ErrInvalidSavedSearchID)
	mockRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
}

func TestSavedSearchService_Delete_RepoError(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	dbErr := errors.New("db down")

	mockRepo.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(false, dbErr)

	err := service.Delete(context.Background(), uuid.New().String(), uuid.New().String())

	assert.ErrorIs(t, err, dbErr)
}
