package mockcallbacks

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/internal/callbacks/models"
	"github.com/stretchr/testify/mock"
)

type MockCallbackRequestRepository struct {
	mock.Mock
}

func (m *MockCallbackRequestRepository) Create(ctx context.Context, request *models.CallbackRequest) (*models.CallbackRequest, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CallbackRequest), args.Error(1)
}
