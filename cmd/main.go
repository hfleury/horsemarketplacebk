package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
)

// enviroment
// database
//

func main() {
	// Configuration
	AppConfig := config.NewVipperService()
	AppConfig.GetAllConfiguration()

	// Logging
	logger := config.NewZerologService()
	ctx := context.Background()

	ctx = logger.WithTrace(ctx, uuid.New().String())

	logger.Log(ctx, config.InfoLevel, "Application started and logging initialized", nil)

	server := gin.New()
	server.Use(middleware.LoggerMiddleware(logger))

	server.GET("/ping", func(c *gin.Context) {
		ctx := c.Request.Context()
		logger.Log(ctx, config.InfoLevel, "Ping endpoint hit", nil)

		c.JSON(http.StatusOK, gin.H{"message": "Pong"})
	})

	if err := server.Run(":8080"); err != nil {
		logger.Log(context.Background(), config.FatalLevel, "Server failed to start", map[string]any{
			"error": err.Error(),
		})
	}

}
