package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/common"
	"github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/hfleury/horsemarketplacebk/internal/products/services"
)

type ProductHandler struct {
	service services.ProductService
	logger  config.Logging
}

func NewProductHandler(service services.ProductService, logger config.Logging) *ProductHandler {
	return &ProductHandler{
		service: service,
		logger:  logger,
	}
}

func (h *ProductHandler) Create(c *gin.Context) {
	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Invalid product payload", map[string]any{"error": err.Error()})
		c.JSON(http.StatusBadRequest, common.NewErrorResponse("Invalid payload"))
		return
	}

	// Get User ID from context (set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return
	}
	product.UserID = uuid.MustParse(userIDStr.(string))

	createdProduct, err := h.service.Create(c.Request.Context(), &product)
	if err != nil {
		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Failed to create product", map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Failed to create product"))
		return
	}

	c.JSON(http.StatusCreated, common.NewSuccessResponse(createdProduct))
}

func (h *ProductHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.NewErrorResponse("Product ID required"))
		return
	}

	product, err := h.service.FindByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Failed to get product", map[string]any{"error": err.Error(), "id": id})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Internal server error"))
		return
	}
	if product == nil {
		c.JSON(http.StatusNotFound, common.NewErrorResponse("Product not found"))
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(product))
}

func (h *ProductHandler) List(c *gin.Context) {
	// Parse query params for search
	query := c.Query("q")
	categoryID := c.Query("category_id")

	// Check for field specific parameters (basic implementation)
	fieldMap := make(map[string]string)
	if model := c.Query("model"); model != "" {
		fieldMap["model"] = model
	}
	if make := c.Query("make"); make != "" {
		fieldMap["make"] = make
	}

	// If no search params, FindAll called inside Search via service logic
	products, err := h.service.Search(c.Request.Context(), query, categoryID, fieldMap)
	if err != nil {
		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Failed to list products", map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Failed to list products"))
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(products))
}

func (h *ProductHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status models.ProductStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.NewErrorResponse("Invalid status"))
		return
	}

	userIDStr, _ := c.Get("user_id")
	// Check role from context
	role, _ := c.Get("role")
	isAdmin := role == "admin" // Assuming "admin" is the role string, check constants if available

	err := h.service.UpdateStatus(c.Request.Context(), id, req.Status, userIDStr.(string), isAdmin)
	if err != nil {
		if err == services.ErrUnauthorized {
			c.JSON(http.StatusForbidden, common.NewErrorResponse(err.Error()))
			return
		}
		if err == services.ErrProductNotFound {
			c.JSON(http.StatusNotFound, common.NewErrorResponse(err.Error()))
			return
		}

		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Failed to update status", map[string]any{"error": err.Error(), "id": id})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Failed to update status"))
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse("Product status updated"))
}

func (h *ProductHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userIDStr, _ := c.Get("user_id")
	role, _ := c.Get("role")
	isAdmin := role == "admin"

	err := h.service.Delete(c.Request.Context(), id, userIDStr.(string), isAdmin)
	if err != nil {
		if err == services.ErrUnauthorized {
			c.JSON(http.StatusForbidden, common.NewErrorResponse(err.Error()))
			return
		}
		if err == services.ErrProductNotFound {
			c.JSON(http.StatusNotFound, common.NewErrorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Failed to delete product"))
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse("Product deleted"))
}
