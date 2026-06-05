package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	"github.com/hfleury/horsemarketplacebk/internal/media"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
	mockemail "github.com/hfleury/horsemarketplacebk/internal/mocks/email"
	productHandlers "github.com/hfleury/horsemarketplacebk/internal/products/handlers"
	productRepos "github.com/hfleury/horsemarketplacebk/internal/products/repositories"
	productServices "github.com/hfleury/horsemarketplacebk/internal/products/services"
	"github.com/hfleury/horsemarketplacebk/internal/router"
	"github.com/hfleury/horsemarketplacebk/internal/system"
	"github.com/hfleury/horsemarketplacebk/internal/tasks"
	"github.com/hfleury/horsemarketplacebk/internal/worker"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

type dbFactory func(config *config.AllConfiguration, logger zerolog.Logger) (*db.PsqlDB, error)

type Server interface {
	Run(addr ...string) error
}

// CombinedRuntime manages the lifetime of all long-running processes
type CombinedRuntime struct {
	ginEngine   *gin.Engine
	asynqServer *asynq.Server
	asynqClient *asynq.Client
	dbPool      *db.PsqlDB
	logger      *config.ZerologService
}

// Run overrides standard blocking logic to safely orchestrate a clean shutdown
func (cr *CombinedRuntime) Run(addr ...string) error {
	address := ":8080"
	if len(addr) > 0 && addr[0] != "" {
		address = addr[0]
	}

	srv := &http.Server{
		Addr:    address,
		Handler: cr.ginEngine,
	}

	// Channel to capture termination signals from the OS
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Start HTTP Server in the background
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			cr.logger.Logger.Fatal().Err(err).Msg("HTTP server crashed")
		}
	}()
	cr.logger.Logger.Info().Msgf("HTTP Server actively listening on %s", address)

	// Block here until Ctrl+C or a SIGTERM is intercepted
	sig := <-stopChan
	cr.logger.Logger.Info().Msgf("Signal %v caught. Graceful shutdown sequence initialized...", sig)

	// Enforce a strict 10-second completion timeout window
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Terminate incoming HTTP traffic
	if err := srv.Shutdown(shutdownCtx); err != nil {
		cr.logger.Logger.Error().Err(err).Msg("HTTP server forced to terminate")
	} else {
		cr.logger.Logger.Info().Msg("HTTP server stopped gracefully")
	}

	// 2. Shut down background workers safely
	cr.asynqServer.Shutdown()
	cr.logger.Logger.Info().Msg("Asynq worker cluster stopped gracefully")

	// 3. Sever connection clients cleanly
	cr.asynqClient.Close()
	cr.logger.Logger.Info().Msg("Asynq client connection pool closed")

	cr.logger.Logger.Info().Msg("All application layers torn down successfully. Exiting.")
	return nil
}

func initializeApp(ctx context.Context, configService config.Configuration, newDB dbFactory) (Server, error) {
	// Configuration
	configService.LoadConfiguration()
	cfg := configService.GetConfig()

	// Logging
	logger := config.NewZerologService()
	logger.Logger.Debug().Msg("Logger initialized")

	// DB PSQL
	database, err := newDB(cfg, *logger.Logger)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Error initializing the Postgres DB")
		return nil, err
	}

	// Add the traceID to the logger
	ctx = logger.WithTrace(ctx, uuid.New().String())
	logger.Log(ctx, config.InfoLevel, "Application started and logging initialized", nil)

	// Repositories
	userRepo := authRepos.NewUserRepoPsql(database, logger)
	sessionRepo := authRepos.NewSessionRepoPsql(database, logger)
	passwordResetRepo := authRepos.NewPasswordResetRepoPsql(database, logger)
	categoryRepo := categoryRepos.NewCategoryRepoPsql(database, logger)
	systemSettingsRepo := system.NewSettingsRepoPsql(database, logger)
	productRepo := productRepos.NewProductRepoPsql(database, logger)

	// Services
	tokenService := services.NewTokenService(cfg, logger)
	userService := services.NewUserService(userRepo, logger, tokenService, sessionRepo)
	userService.SetPasswordResetRepo(passwordResetRepo)
	userService.SetFrontendURL(cfg.FrontendURL)
	categoryService := categoryServices.NewCategoryService(categoryRepo, logger)
	productService := productServices.NewProductService(productRepo, systemSettingsRepo, logger)

	// Handlers
	productHandler := productHandlers.NewProductHandler(productService, logger)

	// Asynq Configuration
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = cfg.Psql.Host + ":6379"
	}

	redisOpt := asynq.RedisClientOpt{Addr: redisAddr}
	asynqClient := asynq.NewClient(redisOpt)
	// Notice: Removed 'defer asynqClient.Close()' to maintain its lifespan for mediaService

	mediaRepo := media.NewPostgresMediaRepository(database.Conn)
	mediaService, err := media.NewMediaService(mediaRepo, asynqClient, cfg)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to initialize media service")
	}

	// Config Asynq Server
	asynqServer := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	mux := asynq.NewServeMux()
	processor, err := worker.NewProcessor(mediaRepo, cfg, logger)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to initialize worker processor")
	} else {
		mux.HandleFunc(tasks.TypeProcessImage, processor.HandleProcessImageTask)
	}

	// Use non-blocking Start instead of Run to hand thread control to our execution manager
	if err := asynqServer.Start(mux); err != nil {
		logger.Logger.Error().Err(err).Msg("Could not start Asynq worker server")
		return nil, err
	}

	// Email verification setup
	emailVerifRepo := authRepos.NewEmailVerificationRepoPsql(database, logger)
	userService.SetEmailVerificationRepo(emailVerifRepo)

	var sender email.Sender
	if cfg.SMTP.Host != "" && cfg.SMTP.Port != "" && cfg.SMTP.From != "" {
		port := 25
		fmt.Sscanf(cfg.SMTP.Port, "%d", &port)
		sender = email.NewSMTPSender(cfg.SMTP.Host, port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
	} else {
		mailgunDomain := os.Getenv("MAILGUN_DOMAIN")
		mailgunAPIKey := os.Getenv("MAILGUN_API_KEY")
		mailFrom := os.Getenv("MAIL_FROM")
		if mailgunDomain != "" && mailgunAPIKey != "" && mailFrom != "" {
			sender = email.NewMailgunSender(mailgunDomain, mailgunAPIKey, mailFrom, 10*time.Second)
		} else {
			sender = mockemail.NewMockSender()
		}
	}
	userService.SetEmailSender(sender)

	// Engine Routing Allocation
	serverEngine := gin.New()
	serverEngine.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5173",
			"http://192.168.49.2:30080",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	serverEngine.Use(middleware.LoggerMiddleware(logger))

	serverEngine = router.SetupRouter(serverEngine, logger, userService, tokenService, categoryService, mediaService, productService, productHandler)

	// Wrap dependencies inside our custom interface coordinator
	return &CombinedRuntime{
		ginEngine:   serverEngine,
		asynqServer: asynqServer,
		asynqClient: asynqClient,
		dbPool:      database,
		logger:      logger,
	}, nil
}

type Launcher struct {
	AppInitializer func(context.Context, config.Configuration, dbFactory) (Server, error)
}

func (l *Launcher) Run(ctx context.Context, configService config.Configuration, newDB dbFactory) error {
	server, err := l.AppInitializer(ctx, configService, newDB)
	if err != nil {
		return err
	}

	// This invokes the CombinedRuntime.Run blocking sequence containing our signal listeners
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
		panic("Application failed to run safely")
	}
}
