package services_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/analytics"
	mockanalytics "github.com/hfleury/horsemarketplacebk/internal/mocks/analytics"
	"github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func eventNames(events []analytics.Event) []string {
	names := make([]string, 0, len(events))
	for _, event := range events {
		names = append(names, event.Name)
	}
	return names
}

func TestCreateProduct_TracksCreatedAndPublishedEvents(t *testing.T) {
	tests := []struct {
		name            string
		requestedStatus models.ProductStatus
		approvalNeeded  bool
		wantEvents      []string
	}{
		{name: "draft", requestedStatus: models.StatusDraft, wantEvents: []string{analytics.EventListingCreated}},
		{name: "publish pending approval", requestedStatus: models.StatusPublished, approvalNeeded: true, wantEvents: []string{analytics.EventListingCreated}},
		{name: "published", requestedStatus: models.StatusPublished, wantEvents: []string{analytics.EventListingCreated, analytics.EventListingPublished}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newPublishTestFixture(config.NewZerologService())
			tracker := mockanalytics.NewFakeTracker()
			fixture.service.SetTracker(tracker)
			fixture.settings.On("IsProductApprovalRequired", mock.Anything).Return(tt.approvalNeeded, nil)
			fixture.notifier.On("NotifyListingPublished", mock.Anything, mock.Anything).Return(nil)
			product := &models.Product{ID: uuid.New(), UserID: uuid.New(), Type: models.TypeHorse, Status: tt.requestedStatus}
			fixture.expectCreateEchoesProduct(product)

			_, err := fixture.service.Create(context.Background(), product)

			require.NoError(t, err)
			events := tracker.Events()
			assert.Equal(t, tt.wantEvents, eventNames(events))
			for _, event := range events {
				assert.Equal(t, product.UserID.String(), event.DistinctID)
				assert.Equal(t, product.ID.String(), event.Properties["listing_id"])
			}
			if len(events) == 2 {
				assert.Equal(t, false, events[1].Properties["by_admin"])
				assert.Equal(t, "published", events[1].Properties["status"])
			}
		})
	}
}

func TestUpdateStatus_TracksListingPublishedForOwnerWithAdminFlag(t *testing.T) {
	fixture := newPublishTestFixture(config.NewZerologService())
	tracker := mockanalytics.NewFakeTracker()
	fixture.service.SetTracker(tracker)
	product := &models.Product{ID: uuid.New(), UserID: uuid.New(), Type: models.TypeHorse, Status: models.StatusPendingApproval}
	fixture.expectExistingProduct(product)
	fixture.repo.On("UpdateStatus", mock.Anything, product.ID.String(), models.StatusPublished).Return(nil)
	fixture.notifier.On("NotifyListingPublished", mock.Anything, product.ID.String()).Return(nil)
	adminID := uuid.New().String()

	err := fixture.service.UpdateStatus(context.Background(), product.ID.String(), models.StatusPublished, adminID, true)

	require.NoError(t, err)
	assert.Equal(t, []analytics.Event{{
		Name:       analytics.EventListingPublished,
		DistinctID: product.UserID.String(),
		Properties: map[string]any{
			"listing_id":   product.ID.String(),
			"listing_type": "horse",
			"status":       "published",
			"by_admin":     true,
		},
	}}, tracker.Events())
	assert.Equal(t, models.StatusPendingApproval, product.Status, "the fetched product must not be mutated")
}

func TestUpdateStatus_NoTransitionIntoPublished_TracksNothing(t *testing.T) {
	fixture := newPublishTestFixture(config.NewZerologService())
	tracker := mockanalytics.NewFakeTracker()
	fixture.service.SetTracker(tracker)
	product := &models.Product{ID: uuid.New(), UserID: uuid.New(), Status: models.StatusPublished}
	fixture.expectExistingProduct(product)
	fixture.repo.On("UpdateStatus", mock.Anything, product.ID.String(), models.StatusPublished).Return(nil)

	err := fixture.service.UpdateStatus(context.Background(), product.ID.String(), models.StatusPublished, "", true)

	require.NoError(t, err)
	assert.Empty(t, tracker.Events())
}
