package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/common"
	"github.com/hfleury/horsemarketplacebk/internal/savedsearches/models"
	"github.com/hfleury/horsemarketplacebk/internal/savedsearches/services"
)

type SavedSearchHandler struct {
	service *services.SavedSearchService
	logger  config.Logging
}

func NewSavedSearchHandler(service *services.SavedSearchService, logger config.Logging) *SavedSearchHandler {
	return &SavedSearchHandler{
		service: service,
		logger:  logger,
	}
}

func (h *SavedSearchHandler) Create(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	req, ok := bindRequest(c)
	if !ok {
		return
	}

	savedSearch, err := h.service.Create(c.Request.Context(), userID, req)
	if err != nil {
		h.respondWithError(c, err, "Failed to create saved search")
		return
	}

	c.JSON(http.StatusCreated, common.NewSuccessResponse(savedSearch))
}

func (h *SavedSearchHandler) List(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	savedSearches, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		h.respondWithError(c, err, "Failed to list saved searches")
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(savedSearches))
}

func (h *SavedSearchHandler) Get(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	savedSearch, err := h.service.Get(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		h.respondWithError(c, err, "Failed to get saved search")
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(savedSearch))
}

func (h *SavedSearchHandler) Update(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	req, ok := bindRequest(c)
	if !ok {
		return
	}

	savedSearch, err := h.service.Update(c.Request.Context(), userID, c.Param("id"), req)
	if err != nil {
		h.respondWithError(c, err, "Failed to update saved search")
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(savedSearch))
}

func (h *SavedSearchHandler) Delete(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		h.respondWithError(c, err, "Failed to delete saved search")
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(nil))
}

func requireUserID(c *gin.Context) (string, bool) {
	value, exists := c.Get("user_id")
	userID, isString := value.(string)
	if !exists || !isString {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return "", false
	}
	return userID, true
}

func bindRequest(c *gin.Context) (models.SavedSearchRequest, bool) {
	var req models.SavedSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.NewErrorResponse("Invalid payload"))
		return req, false
	}
	return req, true
}

// respondWithError uses errors.Is because criteria errors are wrapped with
// the specific rule that failed.
func (h *SavedSearchHandler) respondWithError(c *gin.Context, err error, logMessage string) {
	switch {
	case errors.Is(err, services.ErrInvalidSavedSearchID),
		errors.Is(err, services.ErrInvalidName),
		errors.Is(err, services.ErrInvalidCriteria):
		c.JSON(http.StatusBadRequest, common.NewErrorResponse(err.Error()))
	case errors.Is(err, services.ErrSavedSearchNotFound):
		c.JSON(http.StatusNotFound, common.NewErrorResponse(err.Error()))
	case errors.Is(err, services.ErrSavedSearchLimitReached):
		c.JSON(http.StatusConflict, common.NewErrorResponse(err.Error()))
	default:
		h.logger.Log(c.Request.Context(), config.ErrorLevel, logMessage, map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse(logMessage))
	}
}
