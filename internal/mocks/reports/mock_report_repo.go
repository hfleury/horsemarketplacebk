package mockreports

import (
	"context"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/internal/reports/models"
	"github.com/stretchr/testify/mock"
)

type MockReportRepository struct {
	mock.Mock
}

func (m *MockReportRepository) Create(ctx context.Context, report *models.Report) (*models.Report, error) {
	args := m.Called(ctx, report)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Report), args.Error(1)
}

func (m *MockReportRepository) FindByID(ctx context.Context, id string) (*models.Report, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Report), args.Error(1)
}

func (m *MockReportRepository) FindAll(ctx context.Context, status string) ([]*models.Report, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Report), args.Error(1)
}

func (m *MockReportRepository) UpdateStatus(ctx context.Context, id string, status models.ReportStatus, reviewedBy uuid.UUID) error {
	args := m.Called(ctx, id, status, reviewedBy)
	return args.Error(0)
}
