package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/hfleury/horsemarketplacebk/internal/messaging/handlers"
	messagingServices "github.com/hfleury/horsemarketplacebk/internal/messaging/services"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
)

func registerMessagingRoutes(router *gin.Engine, logger config.Logging, messagingService *messagingServices.MessagingService, tokenService *services.TokenService) {
	messagingHandler := handlers.NewMessagingHandler(messagingService, logger)
	authMiddleware := middleware.NewAuthMiddleware(tokenService, logger)

	v1 := router.Group("/api/v1")

	conversationRoutes := v1.Group("/conversations")
	protected := conversationRoutes.Use(authMiddleware.RequireAuth())
	{
		protected.POST("", messagingHandler.CreateConversation)
		protected.POST("/:id/messages", messagingHandler.SendMessage)
		protected.GET("/:id/messages", messagingHandler.ListMessages)
	}
}
