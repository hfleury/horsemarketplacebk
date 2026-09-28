package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	callbackRequestHandlers "github.com/hfleury/horsemarketplacebk/internal/callbacks/handlers"
	callbackRequestServices "github.com/hfleury/horsemarketplacebk/internal/callbacks/services"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
)

func registerCallbackRequestRoutes(router *gin.Engine, logger config.Logging, callbackRequestService *callbackRequestServices.CallbackRequestService, tokenService *services.TokenService) {
	callbackRequestHandler := callbackRequestHandlers.NewCallbackRequestHandler(callbackRequestService, logger)
	authMiddleware := middleware.NewAuthMiddleware(tokenService, logger)

	v1 := router.Group("/api/v1")

	callbackRequestRoutes := v1.Group("/callback-requests")
	protected := callbackRequestRoutes.Use(authMiddleware.RequireAuth())
	{
		protected.POST("", callbackRequestHandler.Submit)
	}
}
