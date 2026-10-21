package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/analytics"
	"github.com/hfleury/horsemarketplacebk/internal/callbacks/models"
	"github.com/hfleury/horsemarketplacebk/internal/callbacks/services"
	mockanalytics "github.com/hfleury/horsemarketplacebk/internal/mocks/analytics"
	mockcallbacks "github.com/hfleury/horsemarketplacebk/internal/mocks/callbacks"
	mockProducts "github.com/hfleury/horsemarketplacebk/internal/mocks/products"
	productModels "github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSubmit_TracksCallbackRequestedOnlyOnSuccess(t *testing.T) {
	tests := []struct {
		name      string
		createErr error
		tracked   bool
	}{
		{name: "created", tracked: true},
		{name: "create fails", createErr: errors.New("db error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCallbackRepo := new(mockcallbacks.MockCallbackRequestRepository)
			mockProductRepo := new(mockProducts.MockProductRepo)
			service := services.NewCallbackRequestService(mockCallbackRepo, mockProductRepo, nil, config.NewZerologService())
			tracker := mockanalytics.NewFakeTracker()
			service.SetTracker(tracker)
			product := &productModels.Product{ID: uuid.New(), UserID: uuid.New(), Type: productModels.TypeHorse, Status: productModels.StatusPublished}
			buyerID := uuid.New().String()

			mockProductRepo.On("FindByID", mock.Anything, product.ID.String()).Return(product, nil)
			var created *models.CallbackRequest
			if tt.createErr == nil {
				created = &models.CallbackRequest{ID: uuid.New()}
			}
			mockCallbackRepo.On("Create", mock.Anything, mock.Anything).Return(created, tt.createErr)

			_, _ = service.Submit(context.Background(), models.CreateCallbackRequestRequest{ProductID: product.ID.String(), PhoneNumber: "+46 70 123 45 67"}, buyerID)

			if !tt.tracked {
				assert.Empty(t, tracker.Events())
				return
			}
			assert.Equal(t, []analytics.Event{{
				Name:       analytics.EventCallbackRequested,
				DistinctID: buyerID,
				Properties: product.AnalyticsProperties(),
			}}, tracker.Events())
		})
	}
}
