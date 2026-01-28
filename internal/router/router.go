package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	categoryServices "github.com/hfleury/horsemarketplacebk/internal/categories/services"
)

func SetupRouter(router *gin.Engine, logger config.Logging, userService *services.UserService, tokenService *services.TokenService, categoryService *categoryServices.CategoryService) *gin.Engine {
	registerUserRoutes(router, logger, userService, tokenService)
	registerCategoryRoutes(router, logger, categoryService, tokenService)

	return router
}
