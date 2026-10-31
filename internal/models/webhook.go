package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Webhook represents a subscription that receives event notifications
// from FinSight AI. Each tenant can have multiple webhook subscriptions.
// The secret is stored encrypted/hashed by the service layer and never
// exposed via JSON responses.
type Webhook struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TenantID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"tenant_id"`
	URL           string         `gorm:"type:text;not null" json:"url"`
	Secret        string         `gorm:"type:text;not null" json:"-"` // never marshaled
	Events        datatypes.JSON `gorm:"type:jsonb;not null" json:"events"` // JSON array of event names
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	RetryCount    int            `gorm:"default:0" json:"retry_count"`
	LastAttemptAt *time.Time     `json:"last_attempt_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides the default table name used by GORM.
func (Webhook) TableName() string {
	return "webhooks"
}

// BeforeCreate ensures a UUID is set before inserting a new record.
func (w *Webhook) BeforeCreate(tx *gorm.DB) (err error) {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// SetEvents marshals the slice of event names into the JSON column.
func (w *Webhook) SetEvents(events []string) error {
	b, err := json.Marshal(events)
	if err != nil {
		return err
	}
	w.Events = datatypes.JSON(b)
	return nil
}

// GetEvents unmarshals the JSON column back into a slice of event names.
func (w *Webhook) GetEvents() ([]string, error) {
	var ev []string
	if err := json.Unmarshal(w.Events, &ev); err != nil {
		return nil, err
	}
	return ev, nil
}

// SetSecret stores the raw secret. The calling service should encrypt or hash
// the secret before persisting. This method simply assigns the value.
func (w *Webhook) SetSecret(secret string) {
	w.Secret = secret
}

// GetSecret returns the stored secret. It is intended for internal use only
// (e.g., HMAC generation) and must never be exposed in API responses.
func (w *Webhook) GetSecret() string {
	return w.Secret
}