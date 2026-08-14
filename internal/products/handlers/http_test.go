package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	categoriesmodels "github.com/hfleury/horsemarketplacebk/internal/categories/models"
	media "github.com/hfleury/horsemarketplacebk/internal/media"
	mockProducts "github.com/hfleury/horsemarketplacebk/internal/mocks/products"
	"github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestProductHandler(mockService *mockProducts.MockProductService) *ProductHandler {
	return NewProductHandler(mockService, config.NewZerologService())
}

func TestList_DefaultsPageAndLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(mockProducts.MockProductService)
	handler := newTestProductHandler(mockService)

	result := &models.PaginatedProducts{Items: []*models.Product{}, Total: 0, Page: 1, Limit: 20}
	mockService.On("Search", mock.Anything, "", "", map[string]string{}, (*models.HorseFilter)(nil), (*models.LocationFilter)(nil), 1, 20).Return(result, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/products", nil)

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestList_LimitClampedTo100(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(mockProducts.MockProductService)
	handler := newTestProductHandler(mockService)

	result := &models.PaginatedProducts{Items: []*models.Product{}, Total: 0, Page: 1, Limit: 100}
	mockService.On("Search", mock.Anything, "", "", map[string]string{}, (*models.HorseFilter)(nil), (*models.LocationFilter)(nil), 1, 100).Return(result, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/products?limit=500", nil)

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestList_InvalidPageFallsBackToOne(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []string{"0", "-1", "notanumber"}
	for _, pageParam := range cases {
		mockService := new(mockProducts.MockProductService)
		handler := newTestProductHandler(mockService)

		result := &models.PaginatedProducts{Items: []*models.Product{}, Total: 0, Page: 1, Limit: 20}
		mockService.On("Search", mock.Anything, "", "", map[string]string{}, (*models.HorseFilter)(nil), (*models.LocationFilter)(nil), 1, 20).Return(result, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/products?page="+pageParam, nil)

		handler.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	}
}

func TestList_SuccessReturnsPaginatedData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(mockProducts.MockProductService)
	handler := newTestProductHandler(mockService)

	result := &models.PaginatedProducts{
		Items: []*models.Product{{Title: "Test Horse"}},
		Total: 1,
		Page:  1,
		Limit: 20,
	}
	mockService.On("Search", mock.Anything, "", "", map[string]string{}, (*models.HorseFilter)(nil), (*models.LocationFilter)(nil), 1, 20).Return(result, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/products", nil)

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Test Horse")
	assert.Contains(t, w.Body.String(), `"total":1`)
	mockService.AssertExpectations(t)
}

func TestList_LocationFilter_MissingRadiusReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(mockProducts.MockProductService)
	handler := newTestProductHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/products?lat=59.3&lng=18.0", nil)

	handler.List(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "Search")
}

func TestGet_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(mockProducts.MockProductService)
	handler := newTestProductHandler(mockService)

	productID := uuid.New()
	catID := uuid.New()
	catName := "Horses"
	horseName := "Bella"
	product := &models.Product{
		ID:    productID,
		Title: "Test Horse",
		Type:  models.TypeHorse,
		Category: &categoriesmodels.Category{
			Id:   &catID,
			Name: &catName,
		},
		Media: []models.ProductMedia{
			{ProductID: productID, MediaID: uuid.New(), Order: 0, IsPrimary: true, Media: &media.Media{URL: "https://example.com/img.jpg"}},
		},
		Horse: &models.Horse{ProductID: productID, Name: &horseName},
	}
	mockService.On("FindByID", mock.Anything, productID.String()).Return(product, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: productID.String()}}
	c.Request = httptest.NewRequest("GET", "/api/v1/products/"+productID.String(), nil)

	handler.Get(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Test Horse")
	assert.Contains(t, w.Body.String(), "Horses")
	assert.Contains(t, w.Body.String(), "img.jpg")
	assert.Contains(t, w.Body.String(), "Bella")
	mockService.AssertExpectations(t)
}

func TestGet_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(mockProducts.MockProductService)
	handler := newTestProductHandler(mockService)

	productID := uuid.New()
	mockService.On("FindByID", mock.Anything, productID.String()).Return(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: productID.String()}}
	c.Request = httptest.NewRequest("GET", "/api/v1/products/"+productID.String(), nil)

	handler.Get(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestGet_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(mockProducts.MockProductService)
	handler := newTestProductHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/products/", nil)

	handler.Get(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "FindByID")
}

func TestGet_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(mockProducts.MockProductService)
	handler := newTestProductHandler(mockService)

	productID := uuid.New()
	mockService.On("FindByID", mock.Anything, productID.String()).Return(nil, errors.New("db error"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: productID.String()}}
	c.Request = httptest.NewRequest("GET", "/api/v1/products/"+productID.String(), nil)

	handler.Get(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestList_LocationFilter_RadiusOutOfRangeReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(mockProducts.MockProductService)
	handler := newTestProductHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/products?lat=59.3&lng=18.0&radius_km=1000", nil)

	handler.List(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "Search")
}
