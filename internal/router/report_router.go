package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/hfleury/horsemarketplacebk/internal/middleware"
	"github.com/hfleury/horsemarketplacebk/internal/reports/handlers"
	reportServices "github.com/hfleury/horsemarketplacebk/internal/reports/services"
)

func registerReportRoutes(router *gin.Engine, logger config.Logging, reportService *reportServices.ReportService, tokenService *services.TokenService) {
	reportHandler := handlers.NewReportHandler(reportService, logger)
	authMiddleware := middleware.NewAuthMiddleware(tokenService, logger)

	v1 := router.Group("/api/v1")

	reportRoutes := v1.Group("/reports")
	protected := reportRoutes.Use(authMiddleware.RequireAuth())
	{
		protected.POST("", reportHandler.Submit)
	}

	adminRoutes := v1.Group("/admin")
	adminRoutes.Use(authMiddleware.RequireAuth())
	adminRoutes.Use(authMiddleware.RequireRole("admin"))
	{
		adminRoutes.GET("/reports", reportHandler.List)
		adminRoutes.PATCH("/reports/:id/status", reportHandler.Review)
	}
}
