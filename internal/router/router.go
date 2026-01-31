package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	categoryServices "github.com/hfleury/horsemarketplacebk/internal/categories/services"
	"github.com/hfleury/horsemarketplacebk/internal/media"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
)

func SetupRouter(router *gin.Engine, logger config.Logging, userService *services.UserService, tokenService *services.TokenService, categoryService *categoryServices.CategoryService, mediaService *media.MediaService) *gin.Engine {
	router.Use(middleware.CORSMiddleware())
	registerUserRoutes(router, logger, userService, tokenService)
	registerCategoryRoutes(router, logger, categoryService, tokenService)
	registerMediaRoutes(router, logger, mediaService, tokenService)

	return router
}
