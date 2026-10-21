package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/analytics"
	mockanalytics "github.com/hfleury/horsemarketplacebk/internal/mocks/analytics"
	mockProducts "github.com/hfleury/horsemarketplacebk/internal/mocks/products"
	mockReports "github.com/hfleury/horsemarketplacebk/internal/mocks/reports"
	productModels "github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/hfleury/horsemarketplacebk/internal/reports/models"
	"github.com/hfleury/horsemarketplacebk/internal/reports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSubmit_TracksListingReportedOnlyOnSuccess(t *testing.T) {
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
			mockReportRepo := new(mockReports.MockReportRepository)
			mockProductRepo := new(mockProducts.MockProductRepo)
			service := services.NewReportService(mockReportRepo, mockProductRepo, config.NewZerologService())
			tracker := mockanalytics.NewFakeTracker()
			service.SetTracker(tracker)
			productID, reporterID := uuid.New(), uuid.New()

			mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(&productModels.Product{ID: productID, UserID: uuid.New()}, nil)
			var created *models.Report
			if tt.createErr == nil {
				created = &models.Report{ID: uuid.New()}
			}
			mockReportRepo.On("Create", mock.Anything, mock.Anything).Return(created, tt.createErr)

			_, _ = service.Submit(context.Background(), models.CreateReportRequest{ProductID: productID.String(), Reason: string(models.ReasonFraud)}, reporterID.String())

			if !tt.tracked {
				assert.Empty(t, tracker.Events())
				return
			}
			assert.Equal(t, []analytics.Event{{
				Name:       analytics.EventListingReported,
				DistinctID: reporterID.String(),
				Properties: map[string]any{"listing_id": productID.String(), "reason": "fraud"},
			}}, tracker.Events())
		})
	}
}
