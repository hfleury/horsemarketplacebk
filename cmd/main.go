package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/analytics"
	authRepos "github.com/hfleury/horsemarketplacebk/internal/auth/repositories"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	boostTierRepos "github.com/hfleury/horsemarketplacebk/internal/boosttiers/repositories"
	boostTierServices "github.com/hfleury/horsemarketplacebk/internal/boosttiers/services"
	callbackRequestRepos "github.com/hfleury/horsemarketplacebk/internal/callbacks/repositories"
	callbackRequestServices "github.com/hfleury/horsemarketplacebk/internal/callbacks/services"
	categoryRepos "github.com/hfleury/horsemarketplacebk/internal/categories/repositories"
	categoryServices "github.com/hfleury/horsemarketplacebk/internal/categories/services"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/hfleury/horsemarketplacebk/internal/email"
	favoriteRepos "github.com/hfleury/horsemarketplacebk/internal/favorites/repositories"
	favoriteServices "github.com/hfleury/horsemarketplacebk/internal/favorites/services"
	"github.com/hfleury/horsemarketplacebk/internal/geocoding"
	horseAttributeRepos "github.com/hfleury/horsemarketplacebk/internal/horseattributes/repositories"
	horseAttributeServices "github.com/hfleury/horsemarketplacebk/internal/horseattributes/services"
	"github.com/hfleury/horsemarketplacebk/internal/media"
	messagingRepos "github.com/hfleury/horsemarketplacebk/internal/messaging/repositories"
	messagingServices "github.com/hfleury/horsemarketplacebk/internal/messaging/services"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
	mockemail "github.com/hfleury/horsemarketplacebk/internal/mocks/email"
	"github.com/hfleury/horsemarketplacebk/internal/payments/providers"
	paymentRepos "github.com/hfleury/horsemarketplacebk/internal/payments/repositories"
	paymentServices "github.com/hfleury/horsemarketplacebk/internal/payments/services"
	productHandlers "github.com/hfleury/horsemarketplacebk/internal/products/handlers"
	productRepos "github.com/hfleury/horsemarketplacebk/internal/products/repositories"
	productServices "github.com/hfleury/horsemarketplacebk/internal/products/services"
	reportRepos "github.com/hfleury/horsemarketplacebk/internal/reports/repositories"
	reportServices "github.com/hfleury/horsemarketplacebk/internal/reports/services"
	"github.com/hfleury/horsemarketplacebk/internal/router"
	savedSearchRepos "github.com/hfleury/horsemarketplacebk/internal/savedsearches/repositories"
	savedSearchServices "github.com/hfleury/horsemarketplacebk/internal/savedsearches/services"
	"github.com/hfleury/horsemarketplacebk/internal/system"
	"github.com/hfleury/horsemarketplacebk/internal/tasks"
	"github.com/hfleury/horsemarketplacebk/internal/worker"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

const (
	serverAddress           = ":8080"
	serverReadHeaderTimeout = 10 * time.Second
	serverReadTimeout       = 60 * time.Second
	serverWriteTimeout      = 90 * time.Second
	serverIdleTimeout       = 120 * time.Second
	shutdownDrainTimeout    = 10 * time.Second
	emailSendTimeout        = 10 * time.Second
)

type dbFactory func(config *config.AllConfiguration, logger zerolog.Logger) (*db.PsqlDB, error)

type Server interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

func newHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              serverAddress,
		Handler:           handler,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
	}
}

// newResourceReleaser returns the func that stops background work once HTTP
// has drained, so no request can still enqueue tasks, track events or touch the DB.
func newResourceReleaser(asynqServer *asynq.Server, asynqClient *asynq.Client, analyticsClient io.Closer, database *db.PsqlDB, logger *config.ZerologService) func() {
	return func() {
		logger.Logger.Info().Msg("Stopping Asynq worker")
		asynqServer.Shutdown()
		if err := asynqClient.Close(); err != nil {
			logger.Logger.Error().Err(err).Msg("Error closing Asynq client")
		}
		logger.Logger.Info().Msg("Flushing analytics events")
		if err := analyticsClient.Close(); err != nil {
			logger.Logger.Error().Err(err).Msg("Error closing analytics client")
		}
		logger.Logger.Info().Msg("Closing database connection pool")
		_ = database.Close() // Close logs its own error
	}
}

func initializeApp(ctx context.Context, configService config.Configuration, newDB dbFactory) (Server, func(), error) {
	// Configuration
	configService.LoadConfiguration()

	// Logging
	logger := config.NewZerologService()
	logger.Logger.Debug().Msg("Logger initialized")

	// Error tracking
	if err := config.InitSentry(os.Getenv("SENTRY_DSN"), configService.GetConfig().Env); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to initialize Sentry")
	}

	// DB PSQL
	db, err := newDB(configService.GetConfig(), *logger.Logger)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Error initialize the Postgres DB")
		return nil, nil, err
	}

	// Add the traceID to the logger
	ctx = logger.WithTrace(ctx, uuid.New().String())

	logger.Log(ctx, config.InfoLevel, "Application started and logging initialized", nil)

	// Repositories
	userRepo := authRepos.NewUserRepoPsql(db, logger)
	sessionRepo := authRepos.NewSessionRepoPsql(db, logger)
	categoryRepo := categoryRepos.NewCategoryRepoPsql(db, logger)
	systemSettingsRepo := system.NewSettingsRepoPsql(db, logger)
	productRepo := productRepos.NewProductRepoPsql(db, logger)
	horseAttributeRepo := horseAttributeRepos.NewHorseAttributeRepoPsql(db, logger)
	reportRepo := reportRepos.NewReportRepoPsql(db, logger)
	conversationRepo := messagingRepos.NewConversationRepoPsql(db, logger)
	messageRepo := messagingRepos.NewMessageRepoPsql(db, logger)
	callbackRequestRepo := callbackRequestRepos.NewCallbackRequestRepoPsql(db, logger)
	favoriteRepo := favoriteRepos.NewFavoriteRepoPsql(db, logger)
	savedSearchRepo := savedSearchRepos.NewSavedSearchRepoPsql(db, logger)
	boostTierRepo := boostTierRepos.NewBoostTierRepoPsql(db, logger)
	paymentRepo := paymentRepos.NewPaymentRepoPsql(db, logger)

	frontendURL := configService.GetConfig().Frontend.URL

	// Services
	tokenService := services.NewTokenService(configService.GetConfig(), logger)
	userService := services.NewUserService(userRepo, logger, tokenService, sessionRepo)
	categoryService := categoryServices.NewCategoryService(categoryRepo, logger)
	mapboxClient := geocoding.NewMapboxClient(configService.GetConfig().Mapbox.APIKey)
	productService := productServices.NewProductService(productRepo, systemSettingsRepo, logger, mapboxClient)
	horseAttributeService := horseAttributeServices.NewHorseAttributeService(horseAttributeRepo, logger)
	reportService := reportServices.NewReportService(reportRepo, productRepo, logger)
	messagingService := messagingServices.NewMessagingService(conversationRepo, messageRepo, productRepo, userRepo, logger, frontendURL)
	callbackRequestService := callbackRequestServices.NewCallbackRequestService(callbackRequestRepo, productRepo, userRepo, logger)
	favoriteService := favoriteServices.NewFavoriteService(favoriteRepo, productRepo, logger)
	savedSearchService := savedSearchServices.NewSavedSearchService(savedSearchRepo, logger)
	boostTierService := boostTierServices.NewBoostTierService(boostTierRepo, logger)

	// Handlers
	productHandler := productHandlers.NewProductHandler(productService, logger)
	geocodingHandler := geocoding.NewGeocodingHandler(mapboxClient)

	// Asynq Client & Worker
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = configService.GetConfig().Psql.Host + ":6379" // Fallback mostly for local
		// Better to rely on config
	}
	// override if in config map env var
	if val := os.Getenv("REDIS_ADDR"); val != "" {
		redisAddr = val
	}

	redisOpt := asynq.RedisClientOpt{Addr: redisAddr}
	asynqClient := asynq.NewClient(redisOpt)
	productService.SetPublishNotifier(tasks.NewSavedSearchMatchProducer(asynqClient))

	mediaRepo := media.NewPostgresMediaRepository(db.Conn)
	mediaService, err := media.NewMediaService(mediaRepo, asynqClient, configService.GetConfig())
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to initialize media service")
	}

	cfg := configService.GetConfig()
	sender := newEmailSender(cfg)
	paymentService := paymentServices.NewPaymentService(paymentRepo, productRepo, boostTierRepo, newPaymentRegistry(ctx, cfg.Payments, logger), userRepo, frontendURL, logger, time.Now)

	analyticsClient, err := analytics.New(cfg.PostHog, logger)
	if err != nil {
		return nil, nil, err
	}
	userService.SetTracker(analyticsClient)
	productService.SetTracker(analyticsClient)
	messagingService.SetTracker(analyticsClient)
	callbackRequestService.SetTracker(analyticsClient)
	favoriteService.SetTracker(analyticsClient)
	savedSearchService.SetTracker(analyticsClient)
	reportService.SetTracker(analyticsClient)
	paymentService.SetTracker(analyticsClient)

	// Start Asynq Worker Server
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
	processor, err := worker.NewProcessor(mediaRepo, configService.GetConfig(), logger)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to initialize worker processor")
	} else {
		mux.HandleFunc(tasks.TypeProcessImage, processor.HandleProcessImageTask)
	}
	savedSearchAlertService := savedSearchServices.NewSavedSearchAlertService(savedSearchRepo, productRepo, sender, frontendURL, logger)
	mux.HandleFunc(tasks.TypeMatchSavedSearches, worker.NewSavedSearchAlertHandler(savedSearchAlertService).HandleMatchSavedSearchesTask)

	if err := asynqServer.Start(mux); err != nil {
		logger.Logger.Fatal().Err(err).Msg("could not start Asynq worker server")
	}

	// Email verification repository
	emailVerifRepo := authRepos.NewEmailVerificationRepoPsql(db, logger)
	userService.SetEmailVerificationRepo(emailVerifRepo)
	userService.SetBackendURL(cfg.Backend.URL)
	userService.SetEmailSender(sender)
	messagingService.SetEmailSender(sender)
	callbackRequestService.SetEmailSender(sender)
	paymentService.SetEmailSender(sender)

	// Create the Gin router and add middleware
	server := gin.New()
	// Lets gin.Context.Value fall back to the request context, where
	// LoggerMiddleware stores the trace ID that every log line reads.
	server.ContextWithFallback = true
	server.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.Frontend.URL},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	server.Use(middleware.LoggerMiddleware(logger))
	server.Use(sentrygin.New(sentrygin.Options{Repanic: true}))

	// routes
	server = router.SetupRouter(server, logger, userService, tokenService, categoryService, mediaService, productService, productHandler, horseAttributeService, geocodingHandler, reportService, messagingService, callbackRequestService, favoriteService, savedSearchService, boostTierService, paymentService)

	return newHTTPServer(server), newResourceReleaser(asynqServer, asynqClient, analyticsClient, db, logger), nil
}

// newEmailSender picks the sender in order: SMTP config, Mailgun env, then
// an in-memory mock for local development.
func newEmailSender(cfg *config.AllConfiguration) email.Sender {
	if cfg.SMTP.Host != "" && cfg.SMTP.Port != "" && cfg.SMTP.From != "" {
		port := 25
		fmt.Sscanf(cfg.SMTP.Port, "%d", &port)
		return email.NewSMTPSender(cfg.SMTP.Host, port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From, emailSendTimeout)
	}

	mailgunDomain := os.Getenv("MAILGUN_DOMAIN")
	mailgunAPIKey := os.Getenv("MAILGUN_API_KEY")
	mailFrom := os.Getenv("MAIL_FROM")
	if mailgunDomain != "" && mailgunAPIKey != "" && mailFrom != "" {
		return email.NewMailgunSender(mailgunDomain, mailgunAPIKey, mailFrom, emailSendTimeout)
	}

	return mockemail.NewMockSender()
}

// newPaymentRegistry registers the test-only sandbox provider only when it is
// enabled and has a secret, so production gets no provider by default.
func newPaymentRegistry(ctx context.Context, paymentsConfig config.PaymentsConfig, logger config.Logging) *providers.Registry {
	if !paymentsConfig.SandboxEnabled {
		return providers.NewRegistry()
	}
	if paymentsConfig.SandboxSecret == "" {
		logger.Log(ctx, config.WarnLevel, "PAYMENT_SANDBOX_ENABLED is set without PAYMENT_SANDBOX_SECRET; sandbox payments stay disabled", nil)
		return providers.NewRegistry()
	}
	return providers.NewRegistry(providers.NewSandboxProvider(paymentsConfig.SandboxSecret))
}

type Launcher struct {
	AppInitializer func(context.Context, config.Configuration, dbFactory) (Server, func(), error)
}

func (l *Launcher) Run(ctx context.Context, configService config.Configuration, newDB dbFactory) error {
	server, releaseResources, err := l.AppInitializer(ctx, configService, newDB)
	if err != nil {
		return err
	}
	defer releaseResources()

	return serveUntilCancelled(ctx, server)
}

// serveUntilCancelled serves until the server fails or ctx is cancelled, in
// which case in-flight requests get shutdownDrainTimeout to finish.
func serveUntilCancelled(ctx context.Context, server Server) error {
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- server.ListenAndServe()
	}()

	select {
	case err := <-serveResult:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownDrainTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	configService := config.NewVipperService()

	launcher := &Launcher{
		AppInitializer: initializeApp,
	}

	if err := launcher.Run(ctx, configService, db.NewPsqlDB); err != nil {
		fmt.Print(err)
		panic("Application failed")
	}
}
