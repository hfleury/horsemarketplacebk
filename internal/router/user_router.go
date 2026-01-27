package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/handlers"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
)

func registerUserRoutes(router *gin.Engine, logger config.Logging, userService *services.UserService, tokenService *services.TokenService) {
	userHandler := handlers.NewUserHandler(logger, userService)
	authMiddleware := middleware.NewAuthMiddleware(tokenService, logger)

	v1 := router.Group("/api/v1")
	{
		authRoutes := v1.Group("/auth")
		{
			authRoutes.POST("/users", userHandler.CreateUser)
			authRoutes.POST("/resend-verification", userHandler.ResendVerification)
			authRoutes.POST("/login", userHandler.Login)
			authRoutes.POST("/refresh", userHandler.Refresh)
			authRoutes.GET("/verify", userHandler.Verify) // Added verify endpoint mapping if it was missing or just explicit

			// Protected routes
			protected := authRoutes.Group("/")
			protected.Use(authMiddleware.RequireAuth())
			{
				protected.GET("/users", userHandler.GetUserByUsername)
				protected.POST("/logout", userHandler.Logout)
			}
		}
	}
}
