package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/hfleury/horsemarketplacebk/internal/media"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
)

func registerMediaRoutes(router *gin.Engine, logger config.Logging, mediaService *media.MediaService, tokenService *services.TokenService) {
	mediaHandler := media.NewMediaHandler(logger, mediaService)
	authMiddleware := middleware.NewAuthMiddleware(tokenService, logger)

	v1 := router.Group("/api/v1")
	{
		mediaRoutes := v1.Group("/media")
		{
			// Protected routes
			protected := mediaRoutes.Group("")
			protected.Use(authMiddleware.RequireAuth())
			// Optional: Restrict upload to specific roles? For now allowing all authenticated users.
			{
				protected.POST("/upload", mediaHandler.Upload)
			}
		}
	}
}
