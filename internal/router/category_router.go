package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/hfleury/horsemarketplacebk/internal/categories/handlers"
	categoryServices "github.com/hfleury/horsemarketplacebk/internal/categories/services"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
)

func registerCategoryRoutes(router *gin.Engine, logger config.Logging, categoryService *categoryServices.CategoryService, tokenService *services.TokenService) {
	categoryHandler := handlers.NewCategoryHandler(logger, categoryService)
	authMiddleware := middleware.NewAuthMiddleware(tokenService, logger)

	v1 := router.Group("/api/v1")
	{
		catRoutes := v1.Group("/categories")
		{
			// Public routes
			catRoutes.GET("", categoryHandler.GetAllCategories)
			catRoutes.GET("/search", categoryHandler.GetCategoryByName)

			// Admin protected routes
			protected := catRoutes.Group("")
			protected.Use(authMiddleware.RequireAuth())
			protected.Use(authMiddleware.RequireRole("admin"))
			{
				protected.POST("", categoryHandler.CreateCategory)
				protected.PUT("/:id", categoryHandler.UpdateCategory)
				protected.DELETE("/:id", categoryHandler.DeleteCategory)
			}
		}
	}
}
