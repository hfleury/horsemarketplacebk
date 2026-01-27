package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
)

func SetupRouter(router *gin.Engine, logger config.Logging, userService *services.UserService, tokenService *services.TokenService) *gin.Engine {
	registerUserRoutes(router, logger, userService, tokenService)

	return router
}
