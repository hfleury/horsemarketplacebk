package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/hfleury/horsemarketplacebk/internal/boosttiers/handlers"
	boostTierServices "github.com/hfleury/horsemarketplacebk/internal/boosttiers/services"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
)

func registerBoostTierRoutes(router *gin.Engine, logger config.Logging, boostTierService *boostTierServices.BoostTierService, tokenService *services.TokenService) {
	boostTierHandler := handlers.NewBoostTierHandler(boostTierService, logger)
	authMiddleware := middleware.NewAuthMiddleware(tokenService, logger)

	v1 := router.Group("/api/v1")
	v1.GET("/boost-tiers", boostTierHandler.ListActive)

	adminRoutes := v1.Group("/admin/boost-tiers")
	adminRoutes.Use(authMiddleware.RequireAuth())
	adminRoutes.Use(authMiddleware.RequireRole("admin"))
	{
		adminRoutes.GET("", boostTierHandler.ListAll)
		adminRoutes.POST("", boostTierHandler.Create)
		adminRoutes.PUT("/:id", boostTierHandler.Update)
	}
}
