package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/common"
	"github.com/hfleury/horsemarketplacebk/internal/favorites/services"
)

type FavoriteHandler struct {
	service *services.FavoriteService
	logger  config.Logging
}

func NewFavoriteHandler(service *services.FavoriteService, logger config.Logging) *FavoriteHandler {
	return &FavoriteHandler{
		service: service,
		logger:  logger,
	}
}

func (h *FavoriteHandler) Add(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return
	}

	status, err := h.service.Add(c.Request.Context(), userIDStr.(string), c.Param("productId"))
	if err != nil {
		h.respondWithError(c, err, "Failed to add favorite")
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(status))
}

func (h *FavoriteHandler) Remove(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return
	}

	status, err := h.service.Remove(c.Request.Context(), userIDStr.(string), c.Param("productId"))
	if err != nil {
		h.respondWithError(c, err, "Failed to remove favorite")
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(status))
}

func (h *FavoriteHandler) List(c *gin.Context) {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return
	}

	result, err := h.service.List(c.Request.Context(), userIDStr.(string), page, limit)
	if err != nil {
		h.respondWithError(c, err, "Failed to list favorites")
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(result))
}

func (h *FavoriteHandler) ListIDs(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return
	}

	productIDs, err := h.service.ListIDs(c.Request.Context(), userIDStr.(string))
	if err != nil {
		h.respondWithError(c, err, "Failed to list favorite IDs")
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(productIDs))
}

func (h *FavoriteHandler) respondWithError(c *gin.Context, err error, logMessage string) {
	switch err {
	case services.ErrInvalidProductID:
		c.JSON(http.StatusBadRequest, common.NewErrorResponse(err.Error()))
	case services.ErrProductNotFound:
		c.JSON(http.StatusNotFound, common.NewErrorResponse(err.Error()))
	case services.ErrCannotFavoriteOwnListing:
		c.JSON(http.StatusForbidden, common.NewErrorResponse(err.Error()))
	case services.ErrListingNotPublished:
		c.JSON(http.StatusConflict, common.NewErrorResponse(err.Error()))
	default:
		h.logger.Log(c.Request.Context(), config.ErrorLevel, logMessage, map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse(logMessage))
	}
}
