package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
	paymentHandlers "github.com/hfleury/horsemarketplacebk/internal/payments/handlers"
	paymentServices "github.com/hfleury/horsemarketplacebk/internal/payments/services"
)

func registerPaymentRoutes(router *gin.Engine, logger config.Logging, paymentService *paymentServices.PaymentService, tokenService *services.TokenService) {
	paymentHandler := paymentHandlers.NewPaymentHandler(paymentService, logger)
	authMiddleware := middleware.NewAuthMiddleware(tokenService, logger)

	v1 := router.Group("/api/v1")

	// Providers call the webhook server to server, without a user token.
	publicRoutes := v1.Group("/payments")
	publicRoutes.GET("/providers", paymentHandler.ListProviders)
	publicRoutes.POST("/webhooks/:provider", paymentHandler.Webhook)

	protected := v1.Group("/payments")
	protected.Use(authMiddleware.RequireAuth())
	{
		protected.POST("/checkout", paymentHandler.Checkout)
		protected.GET("/:id", paymentHandler.GetStatus)
	}
}
