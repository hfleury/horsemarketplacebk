package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/hfleury/horsemarketplacebk/internal/horseattributes/handlers"
	horseAttributeServices "github.com/hfleury/horsemarketplacebk/internal/horseattributes/services"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
)

func registerHorseAttributeRoutes(router *gin.Engine, logger config.Logging, horseAttributeService *horseAttributeServices.HorseAttributeService, tokenService *services.TokenService) {
	horseAttributeHandler := handlers.NewHorseAttributeHandler(logger, horseAttributeService)
	authMiddleware := middleware.NewAuthMiddleware(tokenService, logger)

	v1 := router.Group("/api/v1")
	{
		attrRoutes := v1.Group("/horse-attributes")
		{
			// Public routes
			attrRoutes.GET("", horseAttributeHandler.List)

			// Admin protected routes
			protected := attrRoutes.Group("")
			protected.Use(authMiddleware.RequireAuth())
			protected.Use(authMiddleware.RequireRole("admin"))
			{
				protected.POST("", horseAttributeHandler.Create)
				protected.DELETE("/:id", horseAttributeHandler.Delete)
			}
		}
	}
}
