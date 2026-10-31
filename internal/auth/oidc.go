package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

// OIDCConfig holds configuration required to initialise an OIDC client.
type OIDCConfig struct {
	// Issuer is the OIDC provider URL (e.g. https://accounts.google.com).
	Issuer string
	// ClientID is the OAuth2 client identifier registered with the provider.
	ClientID string
	// ClientSecret is the OAuth2 client secret (optional for public clients).
	ClientSecret string
	// RedirectURL is the URL the provider will redirect to after authentication.
	RedirectURL string
	// Scopes are additional scopes requested beyond openid, profile and email.
	Scopes []string
}

// OIDCClient encapsulates the OIDC provider, verifier and OAuth2 configuration.
type OIDCClient struct {
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
	ctx          context.Context
}

// OIDCClaims represents the subset of standard OpenID Connect claims used by FinSight AI.
type OIDCClaims struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name,omitempty"`
	Picture       string `json:"picture,omitempty"`
}

// NewOIDCClient creates a fully‑initialised OIDC client.
// It validates the supplied configuration, discovers the provider metadata
// and prepares a verifier that will be used for ID‑token validation.
func NewOIDCClient(parentCtx context.Context, cfg OIDCConfig) (*OIDCClient, error) {
	if strings.TrimSpace(cfg.Issuer) == "" {
		return nil, errors.New("oidc: issuer must be provided")
	}
	if strings.TrimSpace(cfg.ClientID) == "" {
		return nil, errors.New("oidc: client_id must be provided")
	}
	// Use a cancellable context with a reasonable timeout for provider discovery.
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, err
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	// Base scopes required for most providers.
	scopes := []string{oidc.ScopeOpenID, "profile", "email"}
	if len(cfg.Scopes) > 0 {
		scopes = append(scopes, cfg.Scopes...)
	}

	oauth2Cfg := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
	}

	return &OIDCClient{
		provider:     provider,
		verifier:     verifier,
		oauth2Config: oauth2Cfg,
		ctx:          parentCtx,
	}, nil
}

// VerifyIDToken validates the supplied raw ID token and returns the parsed claims.
// The verification respects the provider's JWKS and the client ID configured
// during NewOIDCClient creation.
func (c *OIDCClient) VerifyIDToken(rawIDToken string) (*OIDCClaims, error) {
	if strings.TrimSpace(rawIDToken) == "" {
		return nil, errors.New("oidc: empty id token")
	}
	// Use a short‑lived context to avoid hanging verification.
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}
	var claims OIDCClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, err
	}
	return &claims, nil
}

// GetUserInfo calls the provider's UserInfo endpoint using the supplied access token.
// It returns the same claim structure as VerifyIDToken for convenience.
func (c *OIDCClient) GetUserInfo(accessToken string) (*OIDCClaims, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("oidc: empty access token")
	}
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	userInfo, err := c.provider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
	}))
	if err != nil {
		return nil, err
	}
	var claims OIDCClaims
	if err := userInfo.Claims(&claims); err != nil {
		return nil, err
	}
	return &claims, nil
}

// ClaimsContextKey is the Gin context key under which verified OIDC claims are stored.
const ClaimsContextKey = "oidcClaims"

// Middleware returns a Gin handler that extracts a Bearer token from the
// Authorization header, validates it against the OIDC provider and stores
// the resulting claims in the request context.
// If verification fails, the request is aborted with HTTP 401.
func (c *OIDCClient) Middleware() gin.HandlerFunc {
	return func(g *gin.Context) {
		authHeader := g.GetHeader("Authorization")
		if authHeader == "" {
			abortUnauthorized(g, "missing Authorization header")
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			abortUnauthorized(g, "invalid Authorization header format")
			return
		}
		rawToken := parts[1]

		claims, err := c.VerifyIDToken(rawToken)
		if err != nil {
			abortUnauthorized(g, "invalid token: "+err.Error())
			return
		}
		// Store claims for downstream handlers.
		g.Set(ClaimsContextKey, claims)
		g.Next()
	}
}

// abortUnauthorized sends a JSON error response and aborts the Gin context.
func abortUnauthorized(g *gin.Context, message string) {
	g.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error":   "unauthorized",
		"message": message,
	})
}

// FromContext extracts OIDCClaims from a Gin context.
// The boolean indicates whether the claims were present.
func FromContext(g *gin.Context) (*OIDCClaims, bool) {
	val, ok := g.Get(ClaimsContextKey)
	if !ok {
		return nil, false
	}
	claims, ok := val.(*OIDCClaims)
	return claims, ok
}

// AuthCodeURL builds the URL to which users should be redirected to start the
// authentication flow. The caller should generate a cryptographically‑secure
// state value and pass it in.
func (c *OIDCClient) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	return c.oauth2Config.AuthCodeURL(state, opts...)
}

// ExchangeCode exchanges an authorization code for an OAuth2 token set.
// The returned token contains an ID token (if the provider supplies one) and
// an access token that can be used with GetUserInfo.
func (c *OIDCClient) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	if strings.TrimSpace(code) == "" {
		return nil, errors.New("oidc: empty authorization code")
	}
	// Use a timeout to avoid hanging on network issues.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return c.oauth2Config.Exchange(ctx, code)
}