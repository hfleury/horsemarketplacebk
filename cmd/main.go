package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/repositories"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
	"github.com/hfleury/horsemarketplacebk/internal/router"
)

func initializeApp(ctx context.Context, configService config.Configuration) (*gin.Engine, error) {
	// Configuration
	configService.LoadConfiguration()

	// Logging
	logger := config.NewZerologService()

	// DB PSQL
	db, err := db.NewPsqlDB(configService.GetConfig(), *logger.Logger)
	if err != nil {
		logger.Logger.Fatal().Err(err).Msg("Error initialize the Postgres DB")
		return nil, err
	}

	// Add the traceID to the logger
	ctx = logger.WithTrace(ctx, uuid.New().String())

	logger.Log(ctx, config.InfoLevel, "Application started and logging initialized", nil)

	// Repositories
	userRepo := repositories.NewUserRepoPsql(db, logger)

	// Services
	userService := services.NewUserService(userRepo, logger)

	// Create the Gin router and add middleware
	server := gin.New()
	server.Use(middleware.LoggerMiddleware(logger))

	// routes
	server = router.SetupRouter(logger, userService)

	return server, nil
}

func main() {
	ctx := context.Background()

	configService := config.NewVipperService()

	server, err := initializeApp(ctx, configService)
	if err != nil {
		panic("Failed to initialize application")
	}

	if err := server.Run(":8080"); err != nil {
		fmt.Print(err)
		panic("Failed to start server")
	}
}
