package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/hfleury/horsemarketplacebk/internal/common"
)

type UserHandler struct {
	logger      config.Logging
	userService *services.UserService
}

func NewUserHandler(logger config.Logging, userService *services.UserService) *UserHandler {
	return &UserHandler{
		logger:      logger,
		userService: userService,
	}
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	logger := h.logger.GetLoggerFromContext(c)

	response := common.APIResponse{}
	userRequest := models.UserCreateResquest{}

	if err := c.ShouldBindJSON(&userRequest); err != nil {
		requestBody, _ := c.Get("request_body")
		logger.Log(c, config.ErrorLevel, "Failed to bind request", map[string]any{
			"error":        err.Error(),
			"request_body": requestBody,
		})

		response.Status = "error"
		response.Message = "Invalid request body"

		c.JSON(http.StatusBadRequest, response)
		return
	}

	user, err := h.userService.CreateUser(c.Request.Context(), userRequest)
	if err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to create user", map[string]any{
			"error": err.Error(),
		})

		response.Status = "error"
		response.Message = "Failed to create user"
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response.Status = "success"
	response.Message = "User created"
	response.Data = user

	c.JSON(http.StatusCreated, response)
}

func (h *UserHandler) GetUserByUsername(c *gin.Context) {
	logger := h.logger.GetLoggerFromContext(c)

	response := common.APIResponse{}
	userRequest := models.UserGetRequest{}

	if err := c.ShouldBindJSON(&userRequest); err != nil {
		requestBody, _ := c.Get("request_body")
		logger.Log(c, config.ErrorLevel, "Failed to bind request", map[string]any{
			"error":        err.Error(),
			"request_body": requestBody,
		})

		response.Status = "error"
		response.Message = "Invalid request body"

		c.JSON(http.StatusBadRequest, response)
		return
	}

	user, err := h.userService.SelectUserByUsername(c.Request.Context(), userRequest)
	if err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to get user by username", map[string]any{
			"error": err.Error(),
		})

		response.Status = "error"
		response.Message = "Failed to get user by username"
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response.Status = "success"
	response.Message = "User got by username"
	response.Data = user

	c.JSON(http.StatusCreated, response)
}
