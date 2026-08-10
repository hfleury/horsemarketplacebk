package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
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
	mockService.On("Search", mock.Anything, "", "", map[string]string{}, (*models.HorseFilter)(nil), 1, 20).Return(result, nil)

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
	mockService.On("Search", mock.Anything, "", "", map[string]string{}, (*models.HorseFilter)(nil), 1, 100).Return(result, nil)

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
		mockService.On("Search", mock.Anything, "", "", map[string]string{}, (*models.HorseFilter)(nil), 1, 20).Return(result, nil)

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
	mockService.On("Search", mock.Anything, "", "", map[string]string{}, (*models.HorseFilter)(nil), 1, 20).Return(result, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/products", nil)

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Test Horse")
	assert.Contains(t, w.Body.String(), `"total":1`)
	mockService.AssertExpectations(t)
}
