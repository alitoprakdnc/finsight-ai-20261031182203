package service

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/yourorg/finSight/internal/models"
	"github.com/yourorg/finSight/internal/repository"
	"github.com/yourorg/finSight/internal/util"
)

// WebhookService is responsible for delivering events to registered webhook endpoints.
type WebhookService struct {
	repo       repository.WebhookRepository
	httpClient *http.Client
	logger     *zap.Logger

	// Configuration
	maxAttempts int
	baseBackoff  time.Duration
}

// NewWebhookService creates a new instance of WebhookService.
// If httpClient is nil a default client with sensible timeouts is used.
func NewWebhookService(repo repository.WebhookRepository, httpClient *http.Client, logger *zap.Logger) *WebhookService {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:    90 * time.Second,
			},
		}
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &WebhookService{
		repo:         repo,
		httpClient:   httpClient,
		logger:       logger,
		maxAttempts:  3,
		baseBackoff:  500 * time.Millisecond,
	}
}

// DispatchEvent sends the given payload to all webhooks subscribed to the specified event.
// The payload is marshaled to JSON and signed using the subscriber's secret via HMAC‑SHA256.
// Errors from individual webhook deliveries are logged; the function returns an error
// only if the lookup of subscriptions fails.
func (s *WebhookService) DispatchEvent(ctx context.Context, eventName string, payload interface{}) error {
	// Marshal payload once – all subscribers receive the same JSON body.
	body, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("failed to marshal webhook payload",
			zap.Error(err),
			zap.String("event", eventName))
		return err
	}

	// Retrieve all webhook subscriptions that listen to this event.
	subs, err := s.repo.ListByEvent(ctx, eventName)
	if err != nil {
		s.logger.Error("failed to fetch webhook subscriptions",
			zap.Error(err),
			zap.String("event", eventName))
		return err
	}
	if len(subs) == 0 {
		s.logger.Debug("no webhook subscriptions for event",
			zap.String("event", eventName))
		return nil
	}

	// Deliver to each subscription.
	for _, sub := range subs {
		if sub == nil {
			continue
		}
		if err := s.deliverToSubscriber(ctx, sub, eventName, body); err != nil {
			// Delivery errors are logged inside deliverToSubscriber; continue with others.
			s.logger.Warn("webhook delivery failed",
				zap.String("url", sub.URL),
				zap.String("event", eventName),
				zap.Error(err))
		}
	}
	return nil
}

// deliverToSubscriber performs the HTTP POST with retries and HMAC signing.
func (s *WebhookService) deliverToSubscriber(parentCtx context.Context, sub *models.Webhook, eventName string, body []byte) error {
	// Prepare request body copy for each attempt.
	reqBody := bytes.NewReader(body)

	// Compute HMAC signature using the subscriber's secret.
	signature := s.computeSignature(sub.Secret, body)

	// Build the request once; we will clone it for each retry.
	baseReq, err := http.NewRequestWithContext(parentCtx, http.MethodPost, sub.URL, reqBody)
	if err != nil {
		return err
	}
	baseReq.Header.Set("Content-Type", "application/json")
	baseReq.Header.Set("X-Webhook-Event", eventName)
	baseReq.Header.Set("X-Webhook-Signature", signature)

	// Retry loop.
	for attempt := 1; attempt <= s.maxAttempts; attempt++ {
		// Clone request to reset body for each attempt.
		req := baseReq.Clone(parentCtx)
		// Reset body reader.
		req.Body = io.NopCloser(bytes.NewReader(body))

		resp, err := s.httpClient.Do(req)
		if err != nil {
			s.logger.Error("webhook request error",
				zap.String("url", sub.URL),
				zap.Int("attempt", attempt),
				zap.Error(err))
		} else {
			// Drain and close body to reuse connection.
			_ = resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				s.logger.Info("webhook delivered successfully",
					zap.String("url", sub.URL),
					zap.Int("status", resp.StatusCode),
					zap.String("event", eventName))
				return nil
			}
			s.logger.Warn("webhook responded with non‑2xx status",
				zap.String("url", sub.URL),
				zap.Int("status", resp.StatusCode),
				zap.Int("attempt", attempt))
		}

		// If this was the last attempt, break.
		if attempt == s.maxAttempts {
			break
		}
		// Exponential backoff before next try.
		backoff := s.baseBackoff * time.Duration(attempt)
		select {
		case <-time.After(backoff):
			// continue
		case <-parentCtx.Done():
			return parentCtx.Err()
		}
	}
	return fmt.Errorf("failed to deliver webhook to %s after %d attempts", sub.URL, s.maxAttempts)
}

// computeSignature returns a hex‑encoded HMAC‑SHA256 signature using constant‑time operations.
func (s *WebhookService) computeSignature(secret string, payload []byte) string {
	mac := util.GenerateHMAC([]byte(secret), payload) // returns []byte
	return hex.EncodeToString(mac)
}