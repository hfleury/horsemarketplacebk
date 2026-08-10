package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/common"
	"github.com/hfleury/horsemarketplacebk/internal/horseattributes/models"
	"github.com/hfleury/horsemarketplacebk/internal/horseattributes/services"
)

type HorseAttributeHandler struct {
	logger  config.Logging
	service *services.HorseAttributeService
}

func NewHorseAttributeHandler(logger config.Logging, service *services.HorseAttributeService) *HorseAttributeHandler {
	return &HorseAttributeHandler{
		logger:  logger,
		service: service,
	}
}

func (h *HorseAttributeHandler) Create(c *gin.Context) {
	logger := h.logger.GetLoggerFromContext(c)
	response := common.APIResponse{}
	var req models.CreateHorseAttributeOptionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to bind create horse attribute option request", map[string]any{"error": err.Error()})
		response.Status = "error"
		response.Message = "Invalid request"
		c.JSON(http.StatusBadRequest, response)
		return
	}

	option, err := h.service.CreateOption(c.Request.Context(), req)
	if err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to create horse attribute option", map[string]any{"error": err.Error()})
		response.Status = "error"
		response.Message = err.Error()
		c.JSON(http.StatusBadRequest, response)
		return
	}

	response.Status = "success"
	response.Message = "Horse attribute option created successfully"
	response.Data = option
	c.JSON(http.StatusCreated, response)
}

func (h *HorseAttributeHandler) Delete(c *gin.Context) {
	logger := h.logger.GetLoggerFromContext(c)
	response := common.APIResponse{}
	id := c.Param("id")
	if id == "" {
		response.Status = "error"
		response.Message = "Horse attribute option ID required"
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := h.service.DeleteOption(c.Request.Context(), id); err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to delete horse attribute option", map[string]any{"error": err.Error(), "id": id})
		response.Status = "error"
		response.Message = err.Error()
		c.JSON(http.StatusBadRequest, response)
		return
	}

	response.Status = "success"
	response.Message = "Horse attribute option deleted successfully"
	c.JSON(http.StatusOK, response)
}

func (h *HorseAttributeHandler) List(c *gin.Context) {
	logger := h.logger.GetLoggerFromContext(c)
	response := common.APIResponse{}
	attrType := c.Query("type")

	options, err := h.service.ListOptions(c.Request.Context(), attrType)
	if err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to list horse attribute options", map[string]any{"error": err.Error()})
		response.Status = "error"
		response.Message = "Failed to retrieve horse attribute options"
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response.Status = "success"
	response.Data = options
	c.JSON(http.StatusOK, response)
}
