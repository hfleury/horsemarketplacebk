package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/callbacks/models"
	"github.com/hfleury/horsemarketplacebk/internal/callbacks/services"
	"github.com/hfleury/horsemarketplacebk/internal/common"
)

type CallbackRequestHandler struct {
	service *services.CallbackRequestService
	logger  config.Logging
}

func NewCallbackRequestHandler(service *services.CallbackRequestService, logger config.Logging) *CallbackRequestHandler {
	return &CallbackRequestHandler{
		service: service,
		logger:  logger,
	}
}

func (h *CallbackRequestHandler) Submit(c *gin.Context) {
	var req models.CreateCallbackRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.NewErrorResponse("Invalid payload"))
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return
	}

	callbackRequest, err := h.service.Submit(c.Request.Context(), req, userIDStr.(string))
	if err != nil {
		if err == services.ErrProductNotFound {
			c.JSON(http.StatusNotFound, common.NewErrorResponse(err.Error()))
			return
		}
		if err == services.ErrCannotRequestOwnListing {
			c.JSON(http.StatusForbidden, common.NewErrorResponse(err.Error()))
			return
		}
		if err == services.ErrInvalidPhoneNumber {
			c.JSON(http.StatusBadRequest, common.NewErrorResponse(err.Error()))
			return
		}

		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Failed to submit callback request", map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Failed to submit callback request"))
		return
	}

	c.JSON(http.StatusCreated, common.NewSuccessResponse(callbackRequest))
}
