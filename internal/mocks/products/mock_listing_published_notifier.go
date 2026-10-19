package products

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockListingPublishedNotifier struct {
	mock.Mock
}

func (m *MockListingPublishedNotifier) NotifyListingPublished(ctx context.Context, productID string) error {
	args := m.Called(ctx, productID)
	return args.Error(0)
}
