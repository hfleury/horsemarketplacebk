package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/hfleury/horsemarketplacebk/internal/common"
)

type UserHandler struct {
	logger       config.Logging
	userService  *services.UserService
	tokenService *services.TokenService
}

func NewUserHandler(logger config.Logging, userService *services.UserService, tokenService *services.TokenService) *UserHandler {
	return &UserHandler{
		logger:       logger,
		userService:  userService,
		tokenService: tokenService,
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

	user, err := h.userService.SelectUserByUsername(c.Request.Context(), &userRequest)
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

func (h *UserHandler) Login(c *gin.Context) {
	logger := h.logger.GetLoggerFromContext(c)
	response := common.APIResponse{}
	userRequest := models.UserLogin{}

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

	user, err := h.userService.Login(c.Request.Context(), userRequest)
	if err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to login", map[string]any{
			"error": err.Error(),
		})

		response.Status = "error"
		response.Message = "Invalid credentials"
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	token, err := h.tokenService.CreateToken(*user.Username, 24*time.Hour)
	if err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to create token", map[string]any{
			"error": err.Error(),
		})

		response.Status = "error"
		response.Message = "Failed to create token"
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response.Status = "success"
	response.Message = "Login successful"
	response.Data = map[string]string{
		"token": token,
	}

	c.JSON(http.StatusOK, response)
}
