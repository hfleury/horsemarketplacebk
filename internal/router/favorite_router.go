package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	favoriteHandlers "github.com/hfleury/horsemarketplacebk/internal/favorites/handlers"
	favoriteServices "github.com/hfleury/horsemarketplacebk/internal/favorites/services"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
)

func registerFavoriteRoutes(router *gin.Engine, logger config.Logging, favoriteService *favoriteServices.FavoriteService, tokenService *services.TokenService) {
	favoriteHandler := favoriteHandlers.NewFavoriteHandler(favoriteService, logger)
	authMiddleware := middleware.NewAuthMiddleware(tokenService, logger)

	v1 := router.Group("/api/v1")

	favoriteRoutes := v1.Group("/favorites")
	protected := favoriteRoutes.Use(authMiddleware.RequireAuth())
	{
		protected.GET("", favoriteHandler.List)
		protected.GET("/ids", favoriteHandler.ListIDs)
		protected.POST("/:productId", favoriteHandler.Add)
		protected.DELETE("/:productId", favoriteHandler.Remove)
	}
}
