package router

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"finSight/internal/config"
)

// healthResponse defines the JSON payload returned by the health endpoints.
type healthResponse struct {
	Status  string            `json:"status"`            // "alive", "ready", or "unavailable"
	Details map[string]string `json:"details,omitempty"` // optional per‑dependency status
}

// RegisterHealthRoutes registers the liveness and readiness endpoints.
// It should be called from the main router setup after all dependencies are
// initialised.
func RegisterHealthRoutes(r *gin.Engine, cfg *config.Config) {
	health := r.Group("/api/health")
	health.GET("/live", liveHandler)
	health.GET("/ready", readyHandler(cfg))
}

// liveHandler implements the liveness probe. It simply reports that the
// process is up and responding to HTTP requests.
func liveHandler(c *gin.Context) {
	c.JSON(http.StatusOK, healthResponse{
		Status: "alive",
	})
}

// readyHandler implements the readiness probe. It checks that critical
// external services (PostgreSQL and Redis) are reachable and operational.
// If any check fails the endpoint returns HTTP 503 with details.
func readyHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		details := make(map[string]string)

		// PostgreSQL health check
		if err := cfg.DB.PingContext(ctx); err != nil {
			details["postgres"] = err.Error()
			zap.L().Error("PostgreSQL health check failed", zap.Error(err))
		} else {
			details["postgres"] = "ok"
		}

		// Redis health check
		if err := cfg.Redis.Ping(ctx).Err(); err != nil {
			details["redis"] = err.Error()
			zap.L().Error("Redis health check failed", zap.Error(err))
		} else {
			details["redis"] = "ok"
		}

		// Determine overall readiness
		overallStatus := "ready"
		for _, v := range details {
			if v != "ok" {
				overallStatus = "unavailable"
				break
			}
		}

		// Log the result for observability
		if overallStatus == "ready" {
			zap.L().Info("Readiness probe succeeded", zap.Any("details", details))
			c.JSON(http.StatusOK, healthResponse{
				Status:  overallStatus,
				Details: details,
			})
			return
		}

		zap.L().Warn("Readiness probe failed", zap.Any("details", details))
		c.JSON(http.StatusServiceUnavailable, healthResponse{
			Status:  overallStatus,
			Details: details,
		})
	}
}