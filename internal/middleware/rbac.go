package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"finSight/internal/auth"
	"finSight/internal/util/response"
)

// ClaimsContextKey is the Gin context key where the JWT claims are stored
// by the authentication middleware.
const ClaimsContextKey = "claims"

// RequireRoles returns a Gin middleware that permits access only to users
// whose JWT claims contain at least one of the supplied roles.
// If no roles are supplied the middleware allows every authenticated request.
// It responds with:
//   401 Unauthorized – when no JWT claims are present.
//   403 Forbidden  – when the user lacks the required role(s).
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	// Convert slice to map for O(1) look‑ups.
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		// Retrieve the claims set by the auth middleware.
		val, exists := c.Get(ClaimsContextKey)
		if !exists {
			// No claims – the request is unauthenticated.
			respondUnauthorized(c, "authentication required")
			c.Abort()
			return
		}

		claims, ok := val.(*auth.Claims)
		if !ok || claims == nil {
			// Unexpected type – treat as unauthenticated.
			respondUnauthorized(c, "invalid authentication token")
			c.Abort()
			return
		}

		// If no role restriction is defined, allow the request.
		if len(allowed) == 0 {
			c.Next()
			return
		}

		if hasAllowedRole(claims.Roles, allowed) {
			c.Next()
			return
		}

		// Authorized but not permitted.
		respondForbidden(c, "insufficient permissions")
		c.Abort()
	}
}

// hasAllowedRole checks whether any role in userRoles exists in the allowed map.
func hasAllowedRole(userRoles []string, allowed map[string]struct{}) bool {
	for _, r := range userRoles {
		if _, ok := allowed[r]; ok {
			return true
		}
	}
	return false
}

// respondUnauthorized sends a 401 JSON response using the project's response helper.
func respondUnauthorized(c *gin.Context, message string) {
	logger := zap.L()
	logger.Warn("unauthorized access attempt",
		zap.String("path", c.FullPath()),
		zap.String("method", c.Request.Method),
		zap.String("msg", message),
	)
	c.JSON(http.StatusUnauthorized, response.ErrorResponse{
		Status:  http.StatusUnauthorized,
		Message: "unauthorized",
		Detail:  message,
	})
}

// respondForbidden sends a 403 JSON response using the project's response helper.
func respondForbidden(c *gin.Context, message string) {
	logger := zap.L()
	logger.Warn("forbidden access attempt",
		zap.String("path", c.FullPath()),
		zap.String("method", c.Request.Method),
		zap.String("msg", message),
	)
	c.JSON(http.StatusForbidden, response.ErrorResponse{
		Status:  http.StatusForbidden,
		Message: "forbidden",
		Detail:  message,
	})
}