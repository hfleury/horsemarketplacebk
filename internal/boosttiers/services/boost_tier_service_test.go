package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/boosttiers/models"
	"github.com/hfleury/horsemarketplacebk/internal/boosttiers/services"
	mockboosttiers "github.com/hfleury/horsemarketplacebk/internal/mocks/boosttiers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newBoostTierService() (*services.BoostTierService, *mockboosttiers.MockBoostTierRepository) {
	repo := new(mockboosttiers.MockBoostTierRepository)
	return services.NewBoostTierService(repo, config.NewZerologService()), repo
}

func validCreateRequest() models.CreateBoostTierRequest {
	return models.CreateBoostTierRequest{Code: "premium", Name: "  Premium  ", PriceSEK: 299, DurationDays: 60}
}

func validUpdateRequest(isActive bool) models.UpdateBoostTierRequest {
	return models.UpdateBoostTierRequest{Name: " Boost ", PriceSEK: 59.5, DurationDays: 10, IsActive: &isActive}
}

func TestBoostTierService_Create_Success(t *testing.T) {
	service, repo := newBoostTierService()
	repo.On("Create", mock.Anything, mock.MatchedBy(func(tier *models.BoostTier) bool {
		return tier.ID != uuid.Nil && tier.Code == "premium" && tier.Name == "Premium" &&
			tier.PriceSEK == 299 && tier.DurationDays == 60 && tier.IsActive
	})).Return(true, nil)

	tier, err := service.Create(context.Background(), validCreateRequest())

	require.NoError(t, err)
	assert.Equal(t, "Premium", tier.Name)
	assert.True(t, tier.IsActive)
	repo.AssertExpectations(t)
}

func TestBoostTierService_Create_CodeTaken(t *testing.T) {
	service, repo := newBoostTierService()
	repo.On("Create", mock.Anything, mock.Anything).Return(false, nil)

	_, err := service.Create(context.Background(), validCreateRequest())

	assert.ErrorIs(t, err, services.ErrBoostTierCodeTaken)
}

func TestBoostTierService_Create_InvalidRequestSkipsRepository(t *testing.T) {
	tests := map[string]func(*models.CreateBoostTierRequest){
		"invalid code":      func(r *models.CreateBoostTierRequest) { r.Code = "Premium" },
		"blank name":        func(r *models.CreateBoostTierRequest) { r.Name = " " },
		"zero price":        func(r *models.CreateBoostTierRequest) { r.PriceSEK = 0 },
		"too many decimals": func(r *models.CreateBoostTierRequest) { r.PriceSEK = 1.234 },
		"zero duration":     func(r *models.CreateBoostTierRequest) { r.DurationDays = 0 },
		"duration too long": func(r *models.CreateBoostTierRequest) { r.DurationDays = 366 },
	}

	for name, breakRequest := range tests {
		t.Run(name, func(t *testing.T) {
			service, repo := newBoostTierService()
			req := validCreateRequest()
			breakRequest(&req)

			_, err := service.Create(context.Background(), req)

			assert.ErrorIs(t, err, services.ErrInvalidBoostTier)
			repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}
}

func TestBoostTierService_Create_RepositoryError(t *testing.T) {
	service, repo := newBoostTierService()
	repo.On("Create", mock.Anything, mock.Anything).Return(false, errors.New("db down"))

	_, err := service.Create(context.Background(), validCreateRequest())

	assert.EqualError(t, err, "db down")
}

func TestBoostTierService_Update_Success(t *testing.T) {
	service, repo := newBoostTierService()
	id := uuid.New()
	repo.On("Update", mock.Anything, mock.MatchedBy(func(tier *models.BoostTier) bool {
		return tier.ID == id && tier.Name == "Boost" && tier.PriceSEK == 59.5 && tier.DurationDays == 10 && !tier.IsActive
	})).Return(true, nil)

	tier, err := service.Update(context.Background(), id.String(), validUpdateRequest(false))

	require.NoError(t, err)
	assert.False(t, tier.IsActive)
	repo.AssertExpectations(t)
}

func TestBoostTierService_Update_Errors(t *testing.T) {
	tests := map[string]struct {
		id          string
		req         models.UpdateBoostTierRequest
		repoUpdated *bool
		wantErr     error
	}{
		"invalid id":        {id: "not-a-uuid", req: validUpdateRequest(true), wantErr: services.ErrInvalidBoostTierID},
		"missing is_active": {id: uuid.NewString(), req: models.UpdateBoostTierRequest{Name: "Boost", PriceSEK: 49, DurationDays: 7}, wantErr: services.ErrInvalidBoostTier},
		"invalid price":     {id: uuid.NewString(), req: models.UpdateBoostTierRequest{Name: "Boost", PriceSEK: -1, DurationDays: 7}, wantErr: services.ErrInvalidBoostTier},
		"unknown tier":      {id: uuid.NewString(), req: validUpdateRequest(true), repoUpdated: new(bool), wantErr: services.ErrBoostTierNotFound},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			service, repo := newBoostTierService()
			if tt.repoUpdated != nil {
				repo.On("Update", mock.Anything, mock.Anything).Return(*tt.repoUpdated, nil)
			}

			_, err := service.Update(context.Background(), tt.id, tt.req)

			assert.ErrorIs(t, err, tt.wantErr)
			if tt.repoUpdated == nil {
				repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestBoostTierService_ListsPassThrough(t *testing.T) {
	service, repo := newBoostTierService()
	all := []models.BoostTier{{Code: "boost"}, {Code: "retired"}}
	active := []models.BoostTier{{Code: "boost"}}
	repo.On("ListAll", mock.Anything).Return(all, nil)
	repo.On("ListActive", mock.Anything).Return(active, nil)

	gotAll, errAll := service.ListAll(context.Background())
	gotActive, errActive := service.ListActive(context.Background())

	require.NoError(t, errAll)
	require.NoError(t, errActive)
	assert.Equal(t, all, gotAll)
	assert.Equal(t, active, gotActive)
}
