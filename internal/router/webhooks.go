package router

import (
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"finSight/internal/middleware"
	"finSight/internal/models"
	"finSight/internal/service"
	"finSight/internal/util/hmac"
	"finSight/internal/util/response"
	"finSight/internal/util/validation"
)

// RegisterWebhookRoutes registers all webhook related endpoints.
// It expects a router group that already has authentication middleware applied.
func RegisterWebhookRoutes(rg *gin.RouterGroup, webhookSvc *service.WebhookService) {
	webhooks := rg.Group("/webhooks")
	{
		// Management endpoints – require RBAC (admin or accountant)
		webhooks.POST("/subscriptions", middleware.RequireRoles("admin", "accountant"), createSubscriptionHandler(webhookSvc))
		webhooks.GET("/subscriptions", middleware.RequireRoles("admin", "accountant", "viewer"), listSubscriptionsHandler(webhookSvc))
		webhooks.GET("/subscriptions/:id", middleware.RequireRoles("admin", "accountant", "viewer"), getSubscriptionHandler(webhookSvc))
		webhooks.PUT("/subscriptions/:id", middleware.RequireRoles("admin", "accountant"), updateSubscriptionHandler(webhookSvc))
		webhooks.DELETE("/subscriptions/:id", middleware.RequireRoles("admin"), deleteSubscriptionHandler(webhookSvc))

		// Public callback endpoint – no auth, but HMAC verification
		webhooks.POST("/callback/:id", receiveCallbackHandler(webhookSvc))
	}
}

// ---------------------------------------------------------------------------
// Request / Response payloads
// ---------------------------------------------------------------------------

type createWebhookRequest struct {
	URL    string   `json:"url" validate:"required,url"`
	Secret string   `json:"secret" validate:"required,min=8"`
	Events []string `json:"events" validate:"required,dive,oneof=document_processed document_approved document_rejected"`
	Enabled bool    `json:"enabled"`
}

type updateWebhookRequest struct {
	URL    *string   `json:"url,omitempty" validate:"omitempty,url"`
	Secret *string   `json:"secret,omitempty" validate:"omitempty,min=8"`
	Events *[]string `json:"events,omitempty" validate:"omitempty,dive,oneof=document_processed document_approved document_rejected"`
	Enabled *bool    `json:"enabled,omitempty"`
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func createSubscriptionHandler(svc *service.WebhookService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createWebhookRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, errors.Wrap(err, "invalid request payload"), http.StatusBadRequest)
			return
		}
		if err := validation.ValidateStruct(req); err != nil {
			response.Error(c, err, http.StatusBadRequest)
			return
		}

		tenantID, ok := c.Get("tenant_id")
		if !ok {
			response.Error(c, errors.New("tenant context missing"), http.StatusUnauthorized)
			return
		}

		sub := &models.Webhook{
			ID:       uuid.NewString(),
			TenantID: tenantID.(string),
			URL:      req.URL,
			Secret:   req.Secret,
			Events:   req.Events,
			Enabled:  req.Enabled,
		}

		if err := svc.CreateSubscription(c.Request.Context(), sub); err != nil {
			response.Error(c, errors.Wrap(err, "failed to create webhook subscription"), http.StatusInternalServerError)
			return
		}
		response.Created(c, sub)
	}
}

func listSubscriptionsHandler(svc *service.WebhookService) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := c.Get("tenant_id")
		if !ok {
			response.Error(c, errors.New("tenant context missing"), http.StatusUnauthorized)
			return
		}
		subs, err := svc.ListSubscriptions(c.Request.Context(), tenantID.(string))
		if err != nil {
			response.Error(c, errors.Wrap(err, "failed to list subscriptions"), http.StatusInternalServerError)
			return
		}
		response.Success(c, subs)
	}
}

func getSubscriptionHandler(svc *service.WebhookService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, errors.New("missing subscription id"), http.StatusBadRequest)
			return
		}
		tenantID, ok := c.Get("tenant_id")
		if !ok {
			response.Error(c, errors.New("tenant context missing"), http.StatusUnauthorized)
			return
		}
		sub, err := svc.GetSubscription(c.Request.Context(), tenantID.(string), id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				response.Error(c, err, http.StatusNotFound)
			} else {
				response.Error(c, errors.Wrap(err, "failed to fetch subscription"), http.StatusInternalServerError)
			}
			return
		}
		response.Success(c, sub)
	}
}

func updateSubscriptionHandler(svc *service.WebhookService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, errors.New("missing subscription id"), http.StatusBadRequest)
			return
		}
		var req updateWebhookRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, errors.Wrap(err, "invalid request payload"), http.StatusBadRequest)
			return
		}
		if err := validation.ValidateStruct(req); err != nil {
			response.Error(c, err, http.StatusBadRequest)
			return
		}
		tenantID, ok := c.Get("tenant_id")
		if !ok {
			response.Error(c, errors.New("tenant context missing"), http.StatusUnauthorized)
			return
		}
		sub, err := svc.GetSubscription(c.Request.Context(), tenantID.(string), id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				response.Error(c, err, http.StatusNotFound)
			} else {
				response.Error(c, errors.Wrap(err, "failed to fetch subscription"), http.StatusInternalServerError)
			}
			return
		}
		// Apply updates
		if req.URL != nil {
			sub.URL = *req.URL
		}
		if req.Secret != nil {
			sub.Secret = *req.Secret
		}
		if req.Events != nil {
			sub.Events = *req.Events
		}
		if req.Enabled != nil {
			sub.Enabled = *req.Enabled
		}
		if err := svc.UpdateSubscription(c.Request.Context(), sub); err != nil {
			response.Error(c, errors.Wrap(err, "failed to update subscription"), http.StatusInternalServerError)
			return
		}
		response.Success(c, sub)
	}
}

func deleteSubscriptionHandler(svc *service.WebhookService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, errors.New("missing subscription id"), http.StatusBadRequest)
			return
		}
		tenantID, ok := c.Get("tenant_id")
		if !ok {
			response.Error(c, errors.New("tenant context missing"), http.StatusUnauthorized)
			return
		}
		if err := svc.DeleteSubscription(c.Request.Context(), tenantID.(string), id); err != nil {
			if errors.Is(err, service.ErrNotFound) {
				response.Error(c, err, http.StatusNotFound)
			} else {
				response.Error(c, errors.Wrap(err, "failed to delete subscription"), http.StatusInternalServerError)
			}
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// receiveCallbackHandler validates the HMAC signature and forwards the payload
// to the webhook service for processing. The endpoint is unauthenticated; security
// relies on the shared secret per subscription.
func receiveCallbackHandler(svc *service.WebhookService) gin.HandlerFunc {
	return func(c *gin.Context) {
		subID := c.Param("id")
		if subID == "" {
			response.Error(c, errors.New("missing subscription id"), http.StatusBadRequest)
			return
		}
		// Load subscription (no tenant check – callbacks are public)
		sub, err := svc.GetSubscriptionByID(c.Request.Context(), subID)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				response.Error(c, err, http.StatusNotFound)
			} else {
				response.Error(c, errors.Wrap(err, "failed to load subscription"), http.StatusInternalServerError)
			}
			return
		}
		if !sub.Enabled {
			response.Error(c, errors.New("subscription disabled"), http.StatusForbidden)
			return
		}
		signature := c.GetHeader("X-Signature")
		if signature == "" {
			response.Error(c, errors.New("missing X-Signature header"), http.StatusBadRequest)
			return
		}
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			response.Error(c, errors.Wrap(err, "unable to read request body"), http.StatusBadRequest)
			return
		}
		// Verify HMAC using the stored secret
		if !hmac.VerifySignature(sub.Secret, body, signature) {
			response.Error(c, errors.New("invalid signature"), http.StatusUnauthorized)
			return
		}
		// Forward to service for further handling (e.g., persisting event, triggering downstream logic)
		if err := svc.ProcessIncomingEvent(c.Request.Context(), sub, body); err != nil {
			response.Error(c, errors.Wrap(err, "failed to process incoming webhook"), http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusOK)
	}
}