package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// JSONB is a thin wrapper around json.RawMessage that implements the
// driver.Valuer and sql.Scanner interfaces required by GORM for JSONB columns.
type JSONB json.RawMessage

// Value implements the driver.Valuer interface.
// It returns the JSON as a string for storage in a JSONB column.
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return string(j), nil
}

// Scan implements the sql.Scanner interface.
// It converts the database value back into a JSONB type.
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = JSONB("{}")
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = JSONB(v)
	case string:
		*j = JSONB(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONB", value)
	}
	return nil
}

// AuditLog represents an immutable audit trail entry.
// All fields are deliberately write‑once; the table is protected by
// PostgreSQL triggers (see migrations) to enforce immutability.
type AuditLog struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	TenantID   uuid.UUID `gorm:"type:uuid;not null;index"`                     // multi‑tenant scoping
	UserID     uuid.UUID `gorm:"type:uuid;not null;index"`                     // actor
	Action     string    `gorm:"type:varchar(100);not null;index"`            // e.g. "document.upload"
	Resource   string    `gorm:"type:varchar(100);not null;index"`            // e.g. "document"
	ResourceID uuid.UUID `gorm:"type:uuid;index"`                              // optional target identifier
	Details    JSONB     `gorm:"type:jsonb;not null"`                         // arbitrary structured data
	IP         string    `gorm:"type:inet"`                                   // source IP address
	UserAgent  string    `gorm:"type:text"`                                    // raw user‑agent string
	CreatedAt  time.Time `gorm:"autoCreateTime"`                               // immutable timestamp
}

// TableName overrides the default table name used by GORM.
func (AuditLog) TableName() string {
	return "audit_logs"
}

// NewAuditLog creates a new AuditLog instance ready for persistence.
// The `details` argument can be any struct that can be marshaled to JSON.
// If `details` is nil an empty JSON object is stored.
func NewAuditLog(
	tenantID uuid.UUID,
	userID uuid.UUID,
	action string,
	resource string,
	resourceID uuid.UUID,
	details interface{},
	ip string,
	userAgent string,
) (*AuditLog, error) {
	var d JSONB
	if details != nil {
		b, err := json.Marshal(details)
		if err != nil {
			return nil, fmt.Errorf("marshal audit details: %w", err)
		}
		d = JSONB(b)
	} else {
		d = JSONB("{}")
	}

	return &AuditLog{
		TenantID:   tenantID,
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    d,
		IP:         ip,
		UserAgent:  userAgent,
		CreatedAt:  time.Now().UTC(),
	}, nil
}