package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/internal/analytics"
	mockanalytics "github.com/hfleury/horsemarketplacebk/internal/mocks/analytics"
	"github.com/hfleury/horsemarketplacebk/internal/savedsearches/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSavedSearchService_Create_TracksSavedSearchCreated(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	tracker := mockanalytics.NewFakeTracker()
	service.SetTracker(tracker)
	userID := uuid.New()
	categoryID := uuid.New().String()
	req := models.SavedSearchRequest{Name: "Arabians", Criteria: &models.SearchCriteria{CategoryID: &categoryID}}

	mockRepo.On("CountByUserID", mock.Anything, userID).Return(0, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	savedSearch, err := service.Create(context.Background(), userID.String(), req)

	require.NoError(t, err)
	assert.Equal(t, []analytics.Event{{
		Name:       analytics.EventSavedSearchCreated,
		DistinctID: userID.String(),
		Properties: map[string]any{
			"saved_search_id": savedSearch.ID.String(),
			"alerts_enabled":  true,
			"category_id":     categoryID,
		},
	}}, tracker.Events())
}

func TestSavedSearchService_Create_RepoErrorTracksNothing(t *testing.T) {
	service, mockRepo := newSavedSearchService()
	tracker := mockanalytics.NewFakeTracker()
	service.SetTracker(tracker)
	userID := uuid.New()

	mockRepo.On("CountByUserID", mock.Anything, userID).Return(0, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))

	_, err := service.Create(context.Background(), userID.String(), validRequest())

	assert.Error(t, err)
	assert.Empty(t, tracker.Events())
}
