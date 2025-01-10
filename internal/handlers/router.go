package handlers

import "github.com/gin-gonic/gin"

func SetupRouter() *gin.Engine {
	router := gin.Default()

	registerUserRoutes(router)

	return router
}
