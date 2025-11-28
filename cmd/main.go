package main

import (
	"context"
	"fmt"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/repositories"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
	"github.com/hfleury/horsemarketplacebk/internal/router"
	"github.com/rs/zerolog"
)

type dbFactory func(config *config.AllConfiguration, logger zerolog.Logger) (*db.PsqlDB, error)

type Server interface {
	Run(addr ...string) error
}

func initializeApp(ctx context.Context, configService config.Configuration, newDB dbFactory) (Server, error) {
	// Configuration
	configService.LoadConfiguration()

	// Logging
	logger := config.NewZerologService()
	logger.Logger.Debug().Msg("Logger initialized")

	// DB PSQL
	db, err := newDB(configService.GetConfig(), *logger.Logger)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Error initialize the Postgres DB")
		return nil, err
	}

	// Add the traceID to the logger
	ctx = logger.WithTrace(ctx, uuid.New().String())

	logger.Log(ctx, config.InfoLevel, "Application started and logging initialized", nil)

	// Repositories
	userRepo := repositories.NewUserRepoPsql(db, logger)

	// Services
	tokenService := services.NewTokenService(configService.GetConfig(), logger)
	userService := services.NewUserService(userRepo, logger, tokenService)

	// Create the Gin router and add middleware
	server := gin.New()
	server.Use(cors.Default())
	server.Use(middleware.LoggerMiddleware(logger))

	// routes
	server = router.SetupRouter(server, logger, userService)

	return server, nil
}

type Launcher struct {
	AppInitializer func(context.Context, config.Configuration, dbFactory) (Server, error)
}

func (l *Launcher) Run(ctx context.Context, configService config.Configuration, newDB dbFactory) error {
	server, err := l.AppInitializer(ctx, configService, newDB)
	if err != nil {
		return err
	}

	if err := server.Run(":8080"); err != nil {
		return err
	}

	return nil
}

func main() {
	ctx := context.Background()
	configService := config.NewVipperService()

	launcher := &Launcher{
		AppInitializer: initializeApp,
	}

	if err := launcher.Run(ctx, configService, db.NewPsqlDB); err != nil {
		fmt.Print(err)
		panic("Application failed")
	}
}
