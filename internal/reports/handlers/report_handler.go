package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/common"
	"github.com/hfleury/horsemarketplacebk/internal/reports/models"
	"github.com/hfleury/horsemarketplacebk/internal/reports/services"
)

type ReportHandler struct {
	service *services.ReportService
	logger  config.Logging
}

func NewReportHandler(service *services.ReportService, logger config.Logging) *ReportHandler {
	return &ReportHandler{
		service: service,
		logger:  logger,
	}
}

func (h *ReportHandler) Submit(c *gin.Context) {
	var req models.CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.NewErrorResponse("Invalid payload"))
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return
	}

	report, err := h.service.Submit(c.Request.Context(), req, userIDStr.(string))
	if err != nil {
		if err == services.ErrProductNotFound {
			c.JSON(http.StatusNotFound, common.NewErrorResponse(err.Error()))
			return
		}
		if err == services.ErrCannotReportOwnListing {
			c.JSON(http.StatusForbidden, common.NewErrorResponse(err.Error()))
			return
		}
		if err == services.ErrInvalidReason {
			c.JSON(http.StatusBadRequest, common.NewErrorResponse(err.Error()))
			return
		}

		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Failed to submit report", map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Failed to submit report"))
		return
	}

	c.JSON(http.StatusCreated, common.NewSuccessResponse(report))
}

func (h *ReportHandler) List(c *gin.Context) {
	status := c.Query("status")

	reports, err := h.service.ListReports(c.Request.Context(), status)
	if err != nil {
		if err == services.ErrInvalidReviewStatus {
			c.JSON(http.StatusBadRequest, common.NewErrorResponse(err.Error()))
			return
		}

		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Failed to list reports", map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Failed to list reports"))
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(reports))
}

func (h *ReportHandler) Review(c *gin.Context) {
	id := c.Param("id")

	var req models.ReviewReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.NewErrorResponse("Invalid payload"))
		return
	}

	adminUserIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return
	}

	err := h.service.ReviewReport(c.Request.Context(), id, models.ReportStatus(req.Status), adminUserIDStr.(string))
	if err != nil {
		if err == services.ErrReportNotFound {
			c.JSON(http.StatusNotFound, common.NewErrorResponse(err.Error()))
			return
		}
		if err == services.ErrInvalidReviewStatus {
			c.JSON(http.StatusBadRequest, common.NewErrorResponse(err.Error()))
			return
		}

		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Failed to review report", map[string]any{"error": err.Error(), "id": id})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Failed to review report"))
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse("Report status updated"))
}
