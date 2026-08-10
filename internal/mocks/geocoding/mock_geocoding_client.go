package geocoding

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/internal/geocoding"
	"github.com/stretchr/testify/mock"
)

type MockGeocodingClient struct {
	mock.Mock
}

func (m *MockGeocodingClient) Geocode(ctx context.Context, query string) (*geocoding.Coordinates, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*geocoding.Coordinates), args.Error(1)
}
