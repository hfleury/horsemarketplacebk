package services_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	mockProducts "github.com/hfleury/horsemarketplacebk/internal/mocks/products"
	mockReports "github.com/hfleury/horsemarketplacebk/internal/mocks/reports"
	productModels "github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/hfleury/horsemarketplacebk/internal/reports/models"
	"github.com/hfleury/horsemarketplacebk/internal/reports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSubmit_Success(t *testing.T) {
	mockReportRepo := new(mockReports.MockReportRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewReportService(mockReportRepo, mockProductRepo, logger)

	productID := uuid.New()
	ownerID := uuid.New()
	reporterID := uuid.New()

	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(&productModels.Product{ID: productID, UserID: ownerID}, nil)
	mockReportRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *models.Report) bool {
		return r.ProductID == productID && r.ReporterUserID == reporterID && r.Reason == models.ReasonSpam && r.Status == models.StatusPending
	})).Return(&models.Report{ID: uuid.New(), ProductID: productID, ReporterUserID: reporterID, Reason: models.ReasonSpam, Status: models.StatusPending}, nil)

	req := models.CreateReportRequest{ProductID: productID.String(), Reason: string(models.ReasonSpam)}

	report, err := service.Submit(context.Background(), req, reporterID.String())

	assert.NoError(t, err)
	assert.Equal(t, models.StatusPending, report.Status)
	mockReportRepo.AssertExpectations(t)
	mockProductRepo.AssertExpectations(t)
}

func TestSubmit_InvalidReason(t *testing.T) {
	mockReportRepo := new(mockReports.MockReportRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewReportService(mockReportRepo, mockProductRepo, logger)

	req := models.CreateReportRequest{ProductID: uuid.New().String(), Reason: "not-a-real-reason"}

	_, err := service.Submit(context.Background(), req, uuid.New().String())

	assert.ErrorIs(t, err, services.ErrInvalidReason)
	mockProductRepo.AssertNotCalled(t, "FindByID")
	mockReportRepo.AssertNotCalled(t, "Create")
}

func TestSubmit_ProductNotFound(t *testing.T) {
	mockReportRepo := new(mockReports.MockReportRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewReportService(mockReportRepo, mockProductRepo, logger)

	productID := uuid.New()
	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(nil, nil)

	req := models.CreateReportRequest{ProductID: productID.String(), Reason: string(models.ReasonFraud)}

	_, err := service.Submit(context.Background(), req, uuid.New().String())

	assert.ErrorIs(t, err, services.ErrProductNotFound)
	mockReportRepo.AssertNotCalled(t, "Create")
}

func TestSubmit_CannotReportOwnListing(t *testing.T) {
	mockReportRepo := new(mockReports.MockReportRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewReportService(mockReportRepo, mockProductRepo, logger)

	productID := uuid.New()
	ownerID := uuid.New()

	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(&productModels.Product{ID: productID, UserID: ownerID}, nil)

	req := models.CreateReportRequest{ProductID: productID.String(), Reason: string(models.ReasonDuplicate)}

	_, err := service.Submit(context.Background(), req, ownerID.String())

	assert.ErrorIs(t, err, services.ErrCannotReportOwnListing)
	mockReportRepo.AssertNotCalled(t, "Create")
}

func TestListReports_Success(t *testing.T) {
	mockReportRepo := new(mockReports.MockReportRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewReportService(mockReportRepo, mockProductRepo, logger)

	expected := []*models.Report{{ID: uuid.New(), Status: models.StatusPending}}
	mockReportRepo.On("FindAll", mock.Anything, "pending").Return(expected, nil)

	reports, err := service.ListReports(context.Background(), "pending")

	assert.NoError(t, err)
	assert.Equal(t, expected, reports)
	mockReportRepo.AssertExpectations(t)
}

func TestListReports_InvalidStatusFilter(t *testing.T) {
	mockReportRepo := new(mockReports.MockReportRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewReportService(mockReportRepo, mockProductRepo, logger)

	_, err := service.ListReports(context.Background(), "bogus-status")

	assert.ErrorIs(t, err, services.ErrInvalidReviewStatus)
	mockReportRepo.AssertNotCalled(t, "FindAll")
}

func TestReviewReport_Success(t *testing.T) {
	mockReportRepo := new(mockReports.MockReportRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewReportService(mockReportRepo, mockProductRepo, logger)

	reportID := uuid.New()
	adminID := uuid.New()

	mockReportRepo.On("FindByID", mock.Anything, reportID.String()).Return(&models.Report{ID: reportID, Status: models.StatusPending}, nil)
	mockReportRepo.On("UpdateStatus", mock.Anything, reportID.String(), models.StatusReviewed, adminID).Return(nil)

	err := service.ReviewReport(context.Background(), reportID.String(), models.StatusReviewed, adminID.String())

	assert.NoError(t, err)
	mockReportRepo.AssertExpectations(t)
}

func TestReviewReport_ReportNotFound(t *testing.T) {
	mockReportRepo := new(mockReports.MockReportRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewReportService(mockReportRepo, mockProductRepo, logger)

	reportID := uuid.New()
	mockReportRepo.On("FindByID", mock.Anything, reportID.String()).Return(nil, nil)

	err := service.ReviewReport(context.Background(), reportID.String(), models.StatusDismissed, uuid.New().String())

	assert.ErrorIs(t, err, services.ErrReportNotFound)
	mockReportRepo.AssertNotCalled(t, "UpdateStatus")
}

func TestReviewReport_InvalidStatus(t *testing.T) {
	mockReportRepo := new(mockReports.MockReportRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewReportService(mockReportRepo, mockProductRepo, logger)

	err := service.ReviewReport(context.Background(), uuid.New().String(), models.StatusPending, uuid.New().String())

	assert.ErrorIs(t, err, services.ErrInvalidReviewStatus)
	mockReportRepo.AssertNotCalled(t, "FindByID")
	mockReportRepo.AssertNotCalled(t, "UpdateStatus")
}
