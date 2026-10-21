package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/internal/analytics"
	mockanalytics "github.com/hfleury/horsemarketplacebk/internal/mocks/analytics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFavoriteService_Add_TracksFavoriteAddedOnlyOnSuccess(t *testing.T) {
	tests := []struct {
		name    string
		addErr  error
		tracked bool
	}{
		{name: "added", tracked: true},
		{name: "repo fails", addErr: errors.New("db error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mockFavoriteRepo, mockProductRepo := newFavoriteService()
			tracker := mockanalytics.NewFakeTracker()
			service.SetTracker(tracker)
			productID, sellerID, buyerID := uuid.New(), uuid.New(), uuid.New()
			product := publishedProduct(productID, sellerID)

			mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(product, nil)
			mockFavoriteRepo.On("Add", mock.Anything, buyerID, productID).Return(tt.addErr)

			_, _ = service.Add(context.Background(), buyerID.String(), productID.String())

			if !tt.tracked {
				assert.Empty(t, tracker.Events())
				return
			}
			assert.Equal(t, []analytics.Event{{
				Name:       analytics.EventFavoriteAdded,
				DistinctID: buyerID.String(),
				Properties: product.AnalyticsProperties(),
			}}, tracker.Events())
		})
	}
}
