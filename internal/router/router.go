package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"finSight/internal/config"
	"finSight/internal/middleware"
	"finSight/internal/router/docs"
	"finSight/internal/router/health"
	"finSight/internal/router/webhooks"

	// Document routes are in a separate package
	"finSight/internal/router/documents"
)

// SetupRouter creates and configures the Gin engine with all routes and middleware.
// It returns a fully‑ready *gin.Engine ready to be started by the main application.
func SetupRouter(cfg *config.Config) *gin.Engine {
	// Set Gin mode based on configuration.
	if cfg.Environment == "development" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create a new Gin engine without any default middleware.
	router := gin.New()

	// Global middleware
	router.Use(
		middleware.Logger(),   // structured request/response logging
		middleware.Recovery(), // panic recovery with JSON error payloads
		middleware.Metrics(), // Prometheus metrics endpoint registration
	)

	// -------------------------------------------------------------------------
	// Public endpoints (no authentication)
	// -------------------------------------------------------------------------
	// Health checks – liveness and readiness
	health.RegisterRoutes(router)

	// Swagger / OpenAPI UI – exposed under /api/docs
	docs.RegisterRoutes(router)

	// -------------------------------------------------------------------------
	// Protected API (JWT + RBAC)
	// -------------------------------------------------------------------------
	// All versioned API routes live under /api/v1
	apiV1 := router.Group("/api/v1")
	{
		// JWT authentication middleware – extracts user claims and stores them in the context.
		apiV1.Use(middleware.JWTAuth())

		// Document handling routes
		documents.RegisterRoutes(apiV1)

		// Webhook management routes
		webhooks.RegisterRoutes(apiV1)
	}

	// -------------------------------------------------------------------------
	// Fallback for undefined routes – return JSON 404
	// -------------------------------------------------------------------------
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  http.StatusNotFound,
			"error":   "endpoint not found",
			"message": "The requested URL does not exist on this server.",
		})
	})

	return router
}