package handlers_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/boosttiers/handlers"
	"github.com/hfleury/horsemarketplacebk/internal/boosttiers/models"
	"github.com/hfleury/horsemarketplacebk/internal/boosttiers/services"
	"github.com/stretchr/testify/assert"
)

type fakeBoostTierService struct {
	tiers      []models.BoostTier
	tier       *models.BoostTier
	err        error
	updatedID  string
	lastCreate models.CreateBoostTierRequest
	lastUpdate models.UpdateBoostTierRequest
}

func (f *fakeBoostTierService) ListAll(context.Context) ([]models.BoostTier, error) {
	return f.tiers, f.err
}

func (f *fakeBoostTierService) ListActive(context.Context) ([]models.BoostTier, error) {
	return f.tiers, f.err
}

func (f *fakeBoostTierService) Create(_ context.Context, req models.CreateBoostTierRequest) (*models.BoostTier, error) {
	f.lastCreate = req
	return f.tier, f.err
}

func (f *fakeBoostTierService) Update(_ context.Context, id string, req models.UpdateBoostTierRequest) (*models.BoostTier, error) {
	f.updatedID = id
	f.lastUpdate = req
	return f.tier, f.err
}

func newRouter(service *fakeBoostTierService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewBoostTierHandler(service, config.NewZerologService())
	router := gin.New()
	router.GET("/boost-tiers", handler.ListActive)
	router.GET("/admin/boost-tiers", handler.ListAll)
	router.POST("/admin/boost-tiers", handler.Create)
	router.PUT("/admin/boost-tiers/:id", handler.Update)
	return router
}

func serve(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(method, path, bytes.NewBufferString(body)))
	return recorder
}

func TestBoostTierHandler_ListsReturnEmptyArrayNotNull(t *testing.T) {
	for _, path := range []string{"/boost-tiers", "/admin/boost-tiers"} {
		t.Run(path, func(t *testing.T) {
			recorder := serve(newRouter(&fakeBoostTierService{tiers: []models.BoostTier{}}), http.MethodGet, path, "")

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Contains(t, recorder.Body.String(), `"data":[]`)
		})
	}
}

func TestBoostTierHandler_ListReturnsTiers(t *testing.T) {
	service := &fakeBoostTierService{tiers: []models.BoostTier{{Code: "boost", PriceSEK: 49, DurationDays: 7, IsActive: true}}}

	recorder := serve(newRouter(service), http.MethodGet, "/boost-tiers", "")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"code":"boost"`)
	assert.Contains(t, recorder.Body.String(), `"price_sek":49`)
	assert.Contains(t, recorder.Body.String(), `"duration_days":7`)
}

func TestBoostTierHandler_CreateReturns201WithTier(t *testing.T) {
	service := &fakeBoostTierService{tier: &models.BoostTier{Code: "premium"}}

	recorder := serve(newRouter(service), http.MethodPost, "/admin/boost-tiers",
		`{"code":"premium","name":"Premium","price_sek":299.5,"duration_days":60}`)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"code":"premium"`)
	assert.Equal(t, models.CreateBoostTierRequest{Code: "premium", Name: "Premium", PriceSEK: 299.5, DurationDays: 60}, service.lastCreate)
}

func TestBoostTierHandler_UpdatePassesIDAndBody(t *testing.T) {
	service := &fakeBoostTierService{tier: &models.BoostTier{Code: "boost"}}

	recorder := serve(newRouter(service), http.MethodPut, "/admin/boost-tiers/abc",
		`{"name":"Boost","price_sek":49,"duration_days":7,"is_active":false}`)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "abc", service.updatedID)
	if assert.NotNil(t, service.lastUpdate.IsActive) {
		assert.False(t, *service.lastUpdate.IsActive)
	}
}

func TestBoostTierHandler_MalformedJSONIs400(t *testing.T) {
	tests := map[string]struct{ method, path string }{
		"create": {http.MethodPost, "/admin/boost-tiers"},
		"update": {http.MethodPut, "/admin/boost-tiers/abc"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			recorder := serve(newRouter(&fakeBoostTierService{}), tt.method, tt.path, `{"name":`)

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.Contains(t, recorder.Body.String(), "Invalid payload")
		})
	}
}

func TestBoostTierHandler_MapsServiceErrorsToStatusCodes(t *testing.T) {
	tests := map[string]struct {
		err        error
		wantStatus int
		wantBody   string
	}{
		"validation rule":  {err: fmt.Errorf("%w: name must be between 1 and 50 characters", services.ErrInvalidBoostTier), wantStatus: http.StatusBadRequest, wantBody: "name must be"},
		"invalid id":       {err: services.ErrInvalidBoostTierID, wantStatus: http.StatusBadRequest, wantBody: "invalid boost tier id"},
		"not found":        {err: services.ErrBoostTierNotFound, wantStatus: http.StatusNotFound, wantBody: "not found"},
		"code taken":       {err: services.ErrBoostTierCodeTaken, wantStatus: http.StatusConflict, wantBody: "already exists"},
		"unexpected error": {err: errors.New("db down"), wantStatus: http.StatusInternalServerError, wantBody: "Failed to update boost tier"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			recorder := serve(newRouter(&fakeBoostTierService{err: tt.err}), http.MethodPut, "/admin/boost-tiers/abc",
				`{"name":"Boost","price_sek":49,"duration_days":7,"is_active":true}`)

			assert.Equal(t, tt.wantStatus, recorder.Code)
			assert.Contains(t, recorder.Body.String(), tt.wantBody)
			assert.NotContains(t, recorder.Body.String(), "db down", "internal errors must not leak")
		})
	}
}

func TestBoostTierHandler_ListErrorIs500(t *testing.T) {
	recorder := serve(newRouter(&fakeBoostTierService{err: errors.New("db down")}), http.MethodGet, "/boost-tiers", "")

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Failed to list boost tiers")
}
