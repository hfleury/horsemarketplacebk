package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/categories/models"
	"github.com/hfleury/horsemarketplacebk/internal/categories/services"
	"github.com/hfleury/horsemarketplacebk/internal/common"
)

type CategoryHandler struct {
	logger  config.Logging
	service *services.CategoryService
}

func NewCategoryHandler(logger config.Logging, service *services.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		logger:  logger,
		service: service,
	}
}

func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	logger := h.logger.GetLoggerFromContext(c)
	response := common.APIResponse{}
	var req models.CreateCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to bind create category request", map[string]any{"error": err.Error()})
		response.Status = "error"
		response.Message = "Invalid request"
		c.JSON(http.StatusBadRequest, response)
		return
	}

	cat, err := h.service.CreateCategory(c.Request.Context(), req)
	if err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to create category", map[string]any{"error": err.Error()})
		response.Status = "error"
		response.Message = err.Error()
		c.JSON(http.StatusBadRequest, response)
		return
	}

	response.Status = "success"
	response.Message = "Category created successfully"
	response.Data = cat
	c.JSON(http.StatusCreated, response)
}

func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	logger := h.logger.GetLoggerFromContext(c)
	response := common.APIResponse{}
	id := c.Param("id")
	if id == "" {
		response.Status = "error"
		response.Message = "Category ID required"
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var req models.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to bind update category request", map[string]any{"error": err.Error()})
		response.Status = "error"
		response.Message = "Invalid request"
		c.JSON(http.StatusBadRequest, response)
		return
	}

	cat, err := h.service.UpdateCategory(c.Request.Context(), id, req)
	if err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to update category", map[string]any{"error": err.Error(), "id": id})
		response.Status = "error"
		response.Message = err.Error()
		c.JSON(http.StatusBadRequest, response)
		return
	}

	response.Status = "success"
	response.Message = "Category updated successfully"
	response.Data = cat
	c.JSON(http.StatusOK, response)
}

func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	logger := h.logger.GetLoggerFromContext(c)
	response := common.APIResponse{}
	id := c.Param("id")
	if id == "" {
		response.Status = "error"
		response.Message = "Category ID required"
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := h.service.DeleteCategory(c.Request.Context(), id); err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to delete category", map[string]any{"error": err.Error(), "id": id})
		response.Status = "error"
		response.Message = err.Error()
		c.JSON(http.StatusBadRequest, response)
		return
	}

	response.Status = "success"
	response.Message = "Category deleted successfully"
	c.JSON(http.StatusOK, response)
}

func (h *CategoryHandler) GetAllCategories(c *gin.Context) {
	logger := h.logger.GetLoggerFromContext(c)
	response := common.APIResponse{}

	cats, err := h.service.GetAllCategories(c.Request.Context())
	if err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to get categories", map[string]any{"error": err.Error()})
		response.Status = "error"
		response.Message = "Failed to retrieve categories"
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response.Status = "success"
	response.Data = cats
	c.JSON(http.StatusOK, response)
}

func (h *CategoryHandler) GetCategoryByName(c *gin.Context) {
	logger := h.logger.GetLoggerFromContext(c)
	response := common.APIResponse{}
	name := c.Query("name")

	if name == "" {
		response.Status = "error"
		response.Message = "Name parameter required"
		c.JSON(http.StatusBadRequest, response)
		return
	}

	cat, err := h.service.GetCategoryByName(c.Request.Context(), name)
	if err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to find category by name", map[string]any{"error": err.Error(), "name": name})
		response.Status = "error"
		response.Message = "Failed to find category"
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	if cat == nil {
		response.Status = "error"
		response.Message = "Category not found"
		c.JSON(http.StatusNotFound, response)
		return
	}

	response.Status = "success"
	response.Data = cat
	c.JSON(http.StatusOK, response)
}
