package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/boosttiers/models"
	"github.com/hfleury/horsemarketplacebk/internal/boosttiers/services"
	"github.com/hfleury/horsemarketplacebk/internal/common"
)

// boostTierService is what the handler needs from services.BoostTierService.
type boostTierService interface {
	ListAll(ctx context.Context) ([]models.BoostTier, error)
	ListActive(ctx context.Context) ([]models.BoostTier, error)
	Create(ctx context.Context, req models.CreateBoostTierRequest) (*models.BoostTier, error)
	Update(ctx context.Context, id string, req models.UpdateBoostTierRequest) (*models.BoostTier, error)
}

type BoostTierHandler struct {
	service boostTierService
	logger  config.Logging
}

func NewBoostTierHandler(service boostTierService, logger config.Logging) *BoostTierHandler {
	return &BoostTierHandler{
		service: service,
		logger:  logger,
	}
}

func (h *BoostTierHandler) ListActive(c *gin.Context) {
	tiers, err := h.service.ListActive(c.Request.Context())
	if err != nil {
		h.respondWithError(c, err, "Failed to list boost tiers")
		return
	}
	c.JSON(http.StatusOK, common.NewSuccessResponse(tiers))
}

func (h *BoostTierHandler) ListAll(c *gin.Context) {
	tiers, err := h.service.ListAll(c.Request.Context())
	if err != nil {
		h.respondWithError(c, err, "Failed to list boost tiers")
		return
	}
	c.JSON(http.StatusOK, common.NewSuccessResponse(tiers))
}

func (h *BoostTierHandler) Create(c *gin.Context) {
	var req models.CreateBoostTierRequest
	if !bindRequest(c, &req) {
		return
	}
	tier, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		h.respondWithError(c, err, "Failed to create boost tier")
		return
	}
	c.JSON(http.StatusCreated, common.NewSuccessResponse(tier))
}

func (h *BoostTierHandler) Update(c *gin.Context) {
	var req models.UpdateBoostTierRequest
	if !bindRequest(c, &req) {
		return
	}
	tier, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		h.respondWithError(c, err, "Failed to update boost tier")
		return
	}
	c.JSON(http.StatusOK, common.NewSuccessResponse(tier))
}

func bindRequest(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, common.NewErrorResponse("Invalid payload"))
		return false
	}
	return true
}

// respondWithError uses errors.Is because validation errors are wrapped with
// the specific rule that failed.
func (h *BoostTierHandler) respondWithError(c *gin.Context, err error, logMessage string) {
	switch {
	case errors.Is(err, services.ErrInvalidBoostTier), errors.Is(err, services.ErrInvalidBoostTierID):
		c.JSON(http.StatusBadRequest, common.NewErrorResponse(err.Error()))
	case errors.Is(err, services.ErrBoostTierNotFound):
		c.JSON(http.StatusNotFound, common.NewErrorResponse(err.Error()))
	case errors.Is(err, services.ErrBoostTierCodeTaken):
		c.JSON(http.StatusConflict, common.NewErrorResponse(err.Error()))
	default:
		h.logger.Log(c.Request.Context(), config.ErrorLevel, logMessage, map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse(logMessage))
	}
}
