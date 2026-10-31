package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"finSight/internal/config"
	"finSight/internal/secret"
	"finSight/internal/util/logger"
)

// ---------------------------------------------------------------------
// CustomClaims
// ---------------------------------------------------------------------

// CustomClaims defines the JWT payload used across FinSight AI.
// It embeds jwt.RegisteredClaims for standard fields and adds
// application‑specific data.
type CustomClaims struct {
	TenantID string   `json:"tid"` // tenant identifier
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// ---------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------

// Service provides methods to generate and validate JWT tokens.
type Service struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
	audience   string
	ttl        time.Duration
}

// NewService creates a new JWT Service.
// It loads RSA keys from Google Cloud Secret Manager using the
// provided secret manager implementation and configuration.
func NewService(ctx context.Context, cfg *config.Config, sm secret.Manager) (*Service, error) {
	// Load private key
	privPEM, err := sm.GetSecret(ctx, cfg.JWTPrivateKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve JWT private key secret: %w", err)
	}
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
	}

	// Load public key (optional – can be derived from private key)
	pubPEM, err := sm.GetSecret(ctx, cfg.JWTPublicKeySecret)
	if err != nil {
		// If public key secret is missing we derive it from the private key.
		logger.FromContext(ctx).Warn("public key secret not found, deriving from private key")
	} else {
		pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pubPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA public key: %w", err)
		}
		return &Service{
			privateKey: privKey,
			publicKey:  pubKey,
			issuer:     cfg.JWTIssuer,
			audience:   cfg.JWTAudience,
			ttl:        cfg.JWTTTL,
		}, nil
	}

	// Derive public key from private key if not provided
	pubKey := &privKey.PublicKey

	return &Service{
		privateKey: privKey,
		publicKey:  pubKey,
		issuer:     cfg.JWTIssuer,
		audience:   cfg.JWTAudience,
		ttl:        cfg.JWTTTL,
	}, nil
}

// Generate creates a signed JWT for the supplied user information.
// userID is the subject (sub) claim, tenantID identifies the tenant,
// and roles is a slice of role identifiers (e.g., "admin", "accountant").
func (s *Service) Generate(userID, tenantID string, roles []string) (string, error) {
	now := time.Now().UTC()
	claims := CustomClaims{
		TenantID: tenantID,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			Audience:  []string{s.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(
				now.Add(s.ttl),
			),
			ID: fmt.Sprintf("%s-%d", userID, now.UnixNano()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}
	return signed, nil
}

// Verify parses and validates a JWT string.
// On success it returns the token's CustomClaims.
func (s *Service) Verify(tokenString string) (*CustomClaims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("empty token")
	}
	claims := &CustomClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		// Ensure the signing method is RSA.
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.publicKey, nil
	}, jwt.WithAudience(s.audience), jwt.WithIssuer(s.issuer), jwt.WithLeeway(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("token validation error: %w", err)
	}
	return claims, nil
}

// ---------------------------------------------------------------------
// Gin Middleware
// ---------------------------------------------------------------------

// ClaimsKey is the Gin context key under which the validated claims are stored.
const ClaimsKey = "jwtClaims"

// Middleware returns a Gin handler that validates the Authorization header.
// On success the token's claims are stored in the request context.
// On failure the request is aborted with HTTP 401.
func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		const bearerPrefix = "Bearer "

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, bearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header missing or malformed",
			})
			return
		}

		tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
		claims, err := s.Verify(tokenStr)
		if err != nil {
			logger.FromContext(c.Request.Context()).Warn("JWT verification failed", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			return
		}

		// Store claims for downstream handlers.
		c.Set(ClaimsKey, claims)
		c.Next()
	}
}

// FromContext extracts CustomClaims from a Gin context.
// The second return value indicates whether the claims were present.
func FromContext(c *gin.Context) (*CustomClaims, bool) {
	val, exists := c.Get(ClaimsKey)
	if !exists {
		return nil, false
	}
	claims, ok := val.(*CustomClaims)
	return claims, ok
}

// ---------------------------------------------------------------------
// Helper – token extraction (used by other middlewares if needed)
// ---------------------------------------------------------------------

// ExtractToken extracts a raw JWT string from the Authorization header.
// It returns an empty string if the header is missing or malformed.
func ExtractToken(c *gin.Context) string {
	const bearerPrefix = "Bearer "
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(h, bearerPrefix) {
		return strings.TrimSpace(strings.TrimPrefix(h, bearerPrefix))
	}
	return ""
}