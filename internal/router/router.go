package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
)

func SetupRouter(logger config.Logging, userService *services.UserService) *gin.Engine {
	router := gin.Default()

	registerUserRoutes(router, logger, userService)

	return router
}
