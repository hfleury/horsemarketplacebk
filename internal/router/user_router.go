package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/handlers"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
)

func registerUserRoutes(router *gin.Engine, logger config.Logging, userService *services.UserService) {
	userHandler := handlers.NewUserHandler(logger, userService)

	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/user", userHandler.CreateUser)
		authRoutes.GET("/users", userHandler.GetUserByUsername)
		authRoutes.GET("/login", userHandler.Login)
	}
}
