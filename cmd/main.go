package main

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
)

func main() {
	// Configuration
	AppConfig := config.NewVipperService()
	AppConfig.LoadConfiguration()

	// Logging
	logger := config.NewZerologService()

	// DB PSQL
	db, err := db.NewPsqlDB(AppConfig.Config, *logger.Logger)
	if err != nil {
		logger.Logger.Fatal().Err(err).Msg("Error initialize the Postgres DB")
	}
	defer db.Close()

	ctx := context.Background()

	ctx = logger.WithTrace(ctx, uuid.New().String())

	logger.Log(ctx, config.InfoLevel, "Application started and logging initialized", nil)

	server := gin.New()
	server.Use(middleware.LoggerMiddleware(logger))

	if err := server.Run(":8080"); err != nil {
		logger.Log(context.Background(), config.FatalLevel, "Server failed to start", map[string]any{
			"error": err.Error(),
		})
	}

}
