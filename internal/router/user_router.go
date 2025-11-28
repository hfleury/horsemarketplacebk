package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/handlers"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
)

func registerUserRoutes(router *gin.Engine, logger config.Logging, userService *services.UserService) {
	userHandler := handlers.NewUserHandler(logger, userService)

	v1 := router.Group("/api/v1")
	{
		authRoutes := v1.Group("/auth")
		{
			authRoutes.POST("/users", userHandler.CreateUser)
			authRoutes.GET("/users", userHandler.GetUserByUsername)
			authRoutes.POST("/login", userHandler.Login)
			authRoutes.POST("/logout", userHandler.Logout)
			authRoutes.POST("/refresh", userHandler.Refresh)
		}
	}
}
