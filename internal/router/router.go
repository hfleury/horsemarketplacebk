package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	categoryServices "github.com/hfleury/horsemarketplacebk/internal/categories/services"
	"github.com/hfleury/horsemarketplacebk/internal/media"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
	productHandlers "github.com/hfleury/horsemarketplacebk/internal/products/handlers"
	productServices "github.com/hfleury/horsemarketplacebk/internal/products/services"
)

func SetupRouter(router *gin.Engine, logger config.Logging, userService *services.UserService, tokenService *services.TokenService, categoryService *categoryServices.CategoryService, mediaService *media.MediaService, productService productServices.ProductService, productHandler *productHandlers.ProductHandler) *gin.Engine {
	router.Use(middleware.CORSMiddleware())
	registerUserRoutes(router, logger, userService, tokenService)
	registerCategoryRoutes(router, logger, categoryService, tokenService)
	registerMediaRoutes(router, logger, mediaService, tokenService)
	registerProductRoutes(router, logger, productHandler, tokenService)

	return router
}

func registerProductRoutes(router *gin.Engine, logger config.Logging, handler *productHandlers.ProductHandler, tokenService *services.TokenService) {
	products := router.Group("/products")
	{
		// Public
		products.GET("", handler.List)
		products.GET("/:id", handler.Get)

		// Protected
		authMiddleware := middleware.NewAuthMiddleware(tokenService, logger)
		protected := products.Use(authMiddleware.RequireAuth())
		{
			protected.POST("", handler.Create)
			protected.DELETE("/:id", handler.Delete)
			protected.PATCH("/:id/status", handler.UpdateStatus)
		}
	}
}
