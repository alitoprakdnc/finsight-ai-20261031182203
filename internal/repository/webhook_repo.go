package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/yourorg/finSight/internal/models"
)

// WebhookRepository defines the contract for webhook subscription persistence.
type WebhookRepository interface {
	// Create persists a new webhook subscription.
	Create(ctx context.Context, wh *models.Webhook) error
	// GetByID retrieves a webhook by its primary key.
	GetByID(ctx context.Context, id uint) (*models.Webhook, error)
	// ListByTenant returns all webhooks belonging to a tenant.
	ListByTenant(ctx context.Context, tenantID uint) ([]*models.Webhook, error)
	// Update modifies an existing webhook subscription.
	Update(ctx context.Context, wh *models.Webhook) error
	// Delete removes a webhook subscription.
	Delete(ctx context.Context, id uint) error
	// GetSecretByID fetches only the secret associated with a webhook.
	// This method is used by the webhook service when signing payloads.
	GetSecretByID(ctx context.Context, id uint) (string, error)
}

// webhookRepo is the concrete GORM implementation of WebhookRepository.
type webhookRepo struct {
	db *gorm.DB
}

// NewWebhookRepository creates a new instance of webhookRepo.
func NewWebhookRepository(db *gorm.DB) WebhookRepository {
	return &webhookRepo{db: db}
}

// Create inserts a new webhook record. The caller must ensure that the
// webhook model contains all required fields (validated elsewhere).
func (r *webhookRepo) Create(ctx context.Context, wh *models.Webhook) error {
	if wh == nil {
		return fmt.Errorf("webhook is nil")
	}
	if err := r.db.WithContext(ctx).Create(wh).Error; err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}
	return nil
}

// GetByID fetches a webhook by its ID. Returns gorm.ErrRecordNotFound if missing.
func (r *webhookRepo) GetByID(ctx context.Context, id uint) (*models.Webhook, error) {
	var wh models.Webhook
	if err := r.db.WithContext(ctx).First(&wh, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get webhook id=%d: %w", id, err)
	}
	return &wh, nil
}

// ListByTenant returns all webhook subscriptions for a given tenant.
// The result slice is never nil; it will be empty if no records exist.
func (r *webhookRepo) ListByTenant(ctx context.Context, tenantID uint) ([]*models.Webhook, error) {
	var whs []*models.Webhook
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Find(&whs).Error; err != nil {
		return nil, fmt.Errorf("failed to list webhooks for tenant %d: %w", tenantID, err)
	}
	return whs, nil
}

// Update persists changes to an existing webhook. The primary key must be set.
func (r *webhookRepo) Update(ctx context.Context, wh *models.Webhook) error {
	if wh == nil {
		return fmt.Errorf("webhook is nil")
	}
	if wh.ID == 0 {
		return fmt.Errorf("webhook ID is required for update")
	}
	if err := r.db.WithContext(ctx).Save(wh).Error; err != nil {
		return fmt.Errorf("failed to update webhook id=%d: %w", wh.ID, err)
	}
	return nil
}

// Delete removes a webhook record by ID.
func (r *webhookRepo) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Webhook{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete webhook id=%d: %w", id, err)
	}
	return nil
}

// GetSecretByID retrieves only the secret column for a webhook.
// This avoids loading the whole row when the secret is the only required data.
func (r *webhookRepo) GetSecretByID(ctx context.Context, id uint) (string, error) {
	var secret string
	if err := r.db.WithContext(ctx).
		Model(&models.Webhook{}).
		Select("secret").
		Where("id = ?", id).
		Scan(&secret).Error; err != nil {
		return "", fmt.Errorf("failed to fetch secret for webhook id=%d: %w", id, err)
	}
	return secret, nil
}