package handlers

import (
	"net/http"
	"os"
	"time"

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

	loginResponse, err := h.userService.Login(c.Request.Context(), userRequest)
	if err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to login", map[string]any{
			"error": err.Error(),
		})

		response.Status = "error"
		response.Message = "Invalid credentials"
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	// Put refresh token into a secure, HttpOnly cookie and remove from JSON response
	if loginResponse.RefreshToken != "" {
		// determine secure flag: only true in production
		secure := false
		if os.Getenv("ENVIRONMENT") == "production" {
			secure = true
		}
		// parse expiry
		var expires time.Time
		if loginResponse.RefreshExpiresAt != "" {
			if t, err := time.Parse(time.RFC3339, loginResponse.RefreshExpiresAt); err == nil {
				expires = t
			}
		}
		maxAge := 0
		if !expires.IsZero() {
			maxAge = int(time.Until(expires).Seconds())
			if maxAge < 0 {
				maxAge = 0
			}
		}

		// set cookie on root path so it's available to refresh/logout endpoints
		c.SetCookie("refresh_token", loginResponse.RefreshToken, maxAge, "/", "", secure, true)
		// remove from JSON response to keep it HttpOnly
		loginResponse.RefreshToken = ""
	}

	response.Status = "success"
	response.Message = "Login successful"
	response.Data = loginResponse

	c.JSON(http.StatusOK, response)
}

func (h *UserHandler) Logout(c *gin.Context) {
	logger := h.logger.GetLoggerFromContext(c)
	response := common.APIResponse{}
	var body struct {
		RefreshToken *string `json:"refresh_token"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
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

	// allow refresh token to be provided by cookie or body
	var token string
	if body.RefreshToken != nil && *body.RefreshToken != "" {
		token = *body.RefreshToken
	} else {
		// try cookie
		if cookie, err := c.Cookie("refresh_token"); err == nil {
			token = cookie
		}
	}

	if token == "" {
		response.Status = "error"
		response.Message = "refresh_token required"
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := h.userService.Logout(c.Request.Context(), token); err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to logout", map[string]any{
			"error": err.Error(),
		})
		response.Status = "error"
		response.Message = "Failed to logout"
		response.Error = err.Error()
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// clear cookie on logout
	// set cookie expired
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	response.Status = "success"
	response.Message = "Logged out"
	c.JSON(http.StatusOK, response)
}

func (h *UserHandler) Refresh(c *gin.Context) {
	logger := h.logger.GetLoggerFromContext(c)
	response := common.APIResponse{}

	// Accept refresh token from cookie or POST body
	var token string
	if cookie, err := c.Cookie("refresh_token"); err == nil && cookie != "" {
		token = cookie
	} else {
		var body struct {
			RefreshToken *string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.RefreshToken == nil || *body.RefreshToken == "" {
			response.Status = "error"
			response.Message = "refresh_token required"
			c.JSON(http.StatusBadRequest, response)
			return
		}
		token = *body.RefreshToken
	}

	accessToken, newRefresh, newExpiry, err := h.userService.Refresh(c.Request.Context(), token)
	if err != nil {
		logger.Log(c, config.ErrorLevel, "Failed to refresh token", map[string]any{
			"error": err.Error(),
		})
		response.Status = "error"
		response.Message = "Invalid or expired refresh token"
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	// rotate cookie with new refresh token
	if newRefresh != "" {
		secure := false
		if os.Getenv("ENVIRONMENT") == "production" {
			secure = true
		}
		maxAge := 0
		if newExpiry != "" {
			if t, err := time.Parse(time.RFC3339, newExpiry); err == nil {
				maxAge = int(time.Until(t).Seconds())
				if maxAge < 0 {
					maxAge = 0
				}
			}
		}
		c.SetCookie("refresh_token", newRefresh, maxAge, "/", "", secure, true)
	}

	response.Status = "success"
	response.Message = "Token refreshed"
	response.Data = map[string]string{"access_token": accessToken}
	c.JSON(http.StatusOK, response)
}
