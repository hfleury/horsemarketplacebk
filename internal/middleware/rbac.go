package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/internal/common"
)

func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, common.APIResponse{
				Status:  "error",
				Message: "Unauthorized",
			})
			c.Abort()
			return
		}

		userRole, ok := role.(string)
		if !ok {
			c.JSON(http.StatusForbidden, common.APIResponse{
				Status:  "error",
				Message: "Invalid role format",
			})
			c.Abort()
			return
		}

		// Admin has access to everything
		if userRole == "admin" {
			c.Next()
			return
		}

		if userRole != requiredRole {
			c.JSON(http.StatusForbidden, common.APIResponse{
				Status:  "error",
				Message: "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
