package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/hfleury/horsemarketplacebk/internal/common"
)

type AuthMiddleware struct {
	tokenService *services.TokenService
	logger       config.Logging
}

func NewAuthMiddleware(tokenService *services.TokenService, logger config.Logging) *AuthMiddleware {
	return &AuthMiddleware{
		tokenService: tokenService,
		logger:       logger,
	}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, common.APIResponse{
				Status:  "error",
				Message: "Authorization header required",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, common.APIResponse{
				Status:  "error",
				Message: "Bearer token required",
			})
			c.Abort()
			return
		}

		userID, username, email, role, err := m.tokenService.VerifyToken(tokenString)
		if err != nil {
			m.logger.Log(c, config.ErrorLevel, "Invalid token", map[string]any{
				"error": err.Error(),
			})
			c.JSON(http.StatusUnauthorized, common.APIResponse{
				Status:  "error",
				Message: "Invalid or expired token",
			})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("username", username)
		c.Set("email", email)
		c.Set("role", role)
		c.Next()
	}
}
