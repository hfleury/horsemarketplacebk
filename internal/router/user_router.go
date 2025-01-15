package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/handlers"
)

func registerUserRoutes(router *gin.Engine, logger config.Logging) {
	userHandler := handlers.NewUserHandler(logger)

	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/user", userHandler.CreateUser)
	}
}
