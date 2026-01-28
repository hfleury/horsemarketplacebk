package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	authRepos "github.com/hfleury/horsemarketplacebk/internal/auth/repositories"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	categoryRepos "github.com/hfleury/horsemarketplacebk/internal/categories/repositories"
	categoryServices "github.com/hfleury/horsemarketplacebk/internal/categories/services"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/hfleury/horsemarketplacebk/internal/email"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
	mockemail "github.com/hfleury/horsemarketplacebk/internal/mocks/email"
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
	userRepo := authRepos.NewUserRepoPsql(db, logger)
	sessionRepo := authRepos.NewSessionRepoPsql(db, logger)
	categoryRepo := categoryRepos.NewCategoryRepoPsql(db, logger)

	// Services
	tokenService := services.NewTokenService(configService.GetConfig(), logger)
	userService := services.NewUserService(userRepo, logger, tokenService, sessionRepo)
	categoryService := categoryServices.NewCategoryService(categoryRepo, logger)

	// Email verification repository
	emailVerifRepo := authRepos.NewEmailVerificationRepoPsql(db, logger)
	userService.SetEmailVerificationRepo(emailVerifRepo)
	// Email sender selection (in order): SMTP config, Mailgun env, Mock (dev)
	cfg := configService.GetConfig()
	var sender email.Sender
	if cfg.SMTP.Host != "" && cfg.SMTP.Port != "" && cfg.SMTP.From != "" {
		// parse port
		port := 25
		fmt.Sscanf(cfg.SMTP.Port, "%d", &port)
		sender = email.NewSMTPSender(cfg.SMTP.Host, port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
	} else {
		// fallback to Mailgun if configured via env (keeps previous behavior)
		mailgunDomain := os.Getenv("MAILGUN_DOMAIN")
		mailgunAPIKey := os.Getenv("MAILGUN_API_KEY")
		mailFrom := os.Getenv("MAIL_FROM")
		if mailgunDomain != "" && mailgunAPIKey != "" && mailFrom != "" {
			sender = email.NewMailgunSender(mailgunDomain, mailgunAPIKey, mailFrom, 10*time.Second)
		} else {
			// use centralized mock sender
			sender = mockemail.NewMockSender()
		}
	}
	userService.SetEmailSender(sender)

	// Create the Gin router and add middleware
	server := gin.New()
	server.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	server.Use(middleware.LoggerMiddleware(logger))

	// routes
	server = router.SetupRouter(server, logger, userService, tokenService, categoryService)

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
