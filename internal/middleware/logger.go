package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Logger returns a Gin middleware that logs each request and response using the provided zap.Logger.
// It ensures every request has an X-Request-ID header (generating one if missing) and records
// useful request/response metadata such as method, path, status, latency, client IP and user‑agent.
// Any errors attached to the Gin context are also logged.
func Logger(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Ensure a request ID exists for tracing
		reqID := c.Request.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
			c.Request.Header.Set("X-Request-ID", reqID)
		}
		// Make request ID available downstream and to the client
		c.Set("request_id", reqID)
		c.Writer.Header().Set("X-Request-ID", reqID)

		// Process request
		c.Next()

		// Compute latency
		latency := time.Since(start)

		// Build structured log fields
		fields := []zap.Field{
			zap.String("request_id", reqID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// Append any Gin errors
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				fields = append(fields, zap.String("error", e.Error()))
			}
			logger.Error("request completed with error", fields...)
		} else {
			logger.Info("request completed", fields...)
		}
	}
}