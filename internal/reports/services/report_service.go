package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	productRepositories "github.com/hfleury/horsemarketplacebk/internal/products/repositories"
	"github.com/hfleury/horsemarketplacebk/internal/reports/models"
	"github.com/hfleury/horsemarketplacebk/internal/reports/repositories"
)

var (
	ErrProductNotFound        = errors.New("product not found")
	ErrCannotReportOwnListing = errors.New("cannot report your own listing")
	ErrReportNotFound         = errors.New("report not found")
	ErrInvalidReason          = errors.New("reason must be one of: spam, fraud, inappropriate, duplicate, other")
	ErrInvalidReviewStatus    = errors.New("status must be one of: reviewed, dismissed")
)

var allowedReportReasons = map[string]bool{
	string(models.ReasonSpam):          true,
	string(models.ReasonFraud):         true,
	string(models.ReasonInappropriate): true,
	string(models.ReasonDuplicate):     true,
	string(models.ReasonOther):         true,
}

var allowedReportStatusFilters = map[string]bool{
	string(models.StatusPending):   true,
	string(models.StatusReviewed):  true,
	string(models.StatusDismissed): true,
}

type ReportService struct {
	repo        repositories.ReportRepository
	productRepo productRepositories.ProductRepository
	logger      config.Logging
}

func NewReportService(repo repositories.ReportRepository, productRepo productRepositories.ProductRepository, logger config.Logging) *ReportService {
	return &ReportService{
		repo:        repo,
		productRepo: productRepo,
		logger:      logger,
	}
}

func (s *ReportService) Submit(ctx context.Context, req models.CreateReportRequest, reporterUserID string) (*models.Report, error) {
	if !allowedReportReasons[req.Reason] {
		return nil, ErrInvalidReason
	}

	product, err := s.productRepo.FindByID(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	if product.UserID.String() == reporterUserID {
		return nil, ErrCannotReportOwnListing
	}

	reporterUUID, err := uuid.Parse(reporterUserID)
	if err != nil {
		return nil, err
	}

	report := &models.Report{
		ProductID:      product.ID,
		ReporterUserID: reporterUUID,
		Reason:         models.ReportReason(req.Reason),
		Description:    req.Description,
		Status:         models.StatusPending,
	}

	return s.repo.Create(ctx, report)
}

func (s *ReportService) ListReports(ctx context.Context, status string) ([]*models.Report, error) {
	if status != "" && !allowedReportStatusFilters[status] {
		return nil, ErrInvalidReviewStatus
	}
	return s.repo.FindAll(ctx, status)
}

func (s *ReportService) ReviewReport(ctx context.Context, id string, status models.ReportStatus, adminUserID string) error {
	if status != models.StatusReviewed && status != models.StatusDismissed {
		return ErrInvalidReviewStatus
	}

	r, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if r == nil {
		return ErrReportNotFound
	}

	adminUUID, err := uuid.Parse(adminUserID)
	if err != nil {
		return err
	}

	return s.repo.UpdateStatus(ctx, id, status, adminUUID)
}
