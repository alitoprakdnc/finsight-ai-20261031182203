package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/finSight/finSight/internal/util/logger"
)

// Recovery returns a Gin middleware that recovers from any panics,
// logs the panic with a stack trace, and returns a generic 500 JSON response.
// It should be placed early in the middleware chain (typically right after
// the logger middleware) to catch panics from downstream handlers.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				// Capture stack trace
				stack := debug.Stack()

				// Build structured log fields
				fields := []zap.Field{
					zap.Any("panic", rec),
					zap.ByteString("stack_trace", stack),
				}
				// Include request ID if it was set by the logger middleware
				if rid, exists := c.Get("request_id"); exists {
					if idStr, ok := rid.(string); ok {
						fields = append(fields, zap.String("request_id", idStr))
					}
				}
				// Log the panic
				logger.Get().Error("panic recovered", fields...)

				// Ensure the client receives a clean JSON error payload
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"status": "error",
					"error":  "internal server error",
				})
			}
		}()

		// Continue down the chain
		c.Next()
	}
}