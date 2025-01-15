package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
)

func SetupRouter(logger config.Logging) *gin.Engine {
	router := gin.Default()

	registerUserRoutes(router, logger)

	return router
}
