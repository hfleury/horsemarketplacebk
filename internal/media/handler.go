package media

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/common"
)

type MediaHandler struct {
	logger  config.Logging
	service *MediaService
}

func NewMediaHandler(logger config.Logging, service *MediaService) *MediaHandler {
	return &MediaHandler{
		logger:  logger,
		service: service,
	}
}

func (h *MediaHandler) Upload(c *gin.Context) {
	logger := h.logger.GetLoggerFromContext(c)
	response := common.APIResponse{}

	// Retrieve file from form-data
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		logger.Log(c, config.ErrorLevel, "No file provided", map[string]any{"error": err.Error()})
		response.Status = "error"
		response.Message = "No file provided"
		c.JSON(http.StatusBadRequest, response)
		return
	}
	defer file.Close()

	// Check size (optional limit, e.g. 5MB)
	if header.Size > 5*1024*1024 {
		response.Status = "error"
		response.Message = "File too large (max 5MB)"
		c.JSON(http.StatusBadRequest, response)
		return
	}

	media, err := h.service.UploadFile(c.Request.Context(), file, header)
	if err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to upload file", map[string]any{"error": err.Error()})
		response.Status = "error"
		response.Message = "Failed to upload file"
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response.Status = "success"
	response.Message = "File uploaded successfully"
	response.Data = media
	c.JSON(http.StatusCreated, response)
}
