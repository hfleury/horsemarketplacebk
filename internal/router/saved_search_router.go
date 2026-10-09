package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
	savedSearchHandlers "github.com/hfleury/horsemarketplacebk/internal/savedsearches/handlers"
	savedSearchServices "github.com/hfleury/horsemarketplacebk/internal/savedsearches/services"
)

func registerSavedSearchRoutes(router *gin.Engine, logger config.Logging, savedSearchService *savedSearchServices.SavedSearchService, tokenService *services.TokenService) {
	savedSearchHandler := savedSearchHandlers.NewSavedSearchHandler(savedSearchService, logger)
	authMiddleware := middleware.NewAuthMiddleware(tokenService, logger)

	v1 := router.Group("/api/v1")

	savedSearchRoutes := v1.Group("/saved-searches")
	protected := savedSearchRoutes.Use(authMiddleware.RequireAuth())
	{
		protected.POST("", savedSearchHandler.Create)
		protected.GET("", savedSearchHandler.List)
		protected.GET("/:id", savedSearchHandler.Get)
		protected.PUT("/:id", savedSearchHandler.Update)
		protected.DELETE("/:id", savedSearchHandler.Delete)
	}
}
