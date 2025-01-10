package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
)

func LoggerMiddleware(logger *config.ZerologService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate a trace ID for the request
		traceID := uuid.New().String()

		// Add the logger with trace ID to the context
		ctx := logger.WithTrace(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		// Log the request start
		logger.Log(ctx, config.InfoLevel, "Request started", map[string]any{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
		})

		// Process the request
		c.Next()

		// Log the request end
		logger.Log(ctx, config.InfoLevel, "Request completed", map[string]any{
			"status_code": c.Writer.Status(),
		})
	}
}
