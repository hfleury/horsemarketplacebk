package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/internal/auth/handlers"
)

func registerUserRoutes(router *gin.Engine) {
	userHandler := handlers.NewUserHandler()

	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/user", userHandler.CreateUser)
	}
}
