package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// DocumentStatus represents the processing lifecycle of a document.
type DocumentStatus string

const (
	// DocumentStatusUploaded indicates the file has been uploaded but not yet processed.
	DocumentStatusUploaded DocumentStatus = "uploaded"
	// DocumentStatusProcessing indicates OCR/LLM processing is in progress.
	DocumentStatusProcessing DocumentStatus = "processing"
	// DocumentStatusCompleted indicates processing succeeded and data is extracted.
	DocumentStatusCompleted DocumentStatus = "completed"
	// DocumentStatusFailed indicates processing failed; see ErrorMessage for details.
	DocumentStatusFailed DocumentStatus = "failed"
)

// ExtractedData holds the JSON payload returned by the LLM after parsing OCR text.
// It is stored as a PostgreSQL jsonb column.
type ExtractedData json.RawMessage

// Scan implements the sql.Scanner interface for ExtractedData.
func (e *ExtractedData) Scan(value interface{}) error {
	if value == nil {
		*e = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*e = ExtractedData(v)
		return nil
	case string:
		*e = ExtractedData([]byte(v))
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into ExtractedData", value)
	}
}

// Value implements the driver.Valuer interface for ExtractedData.
func (e ExtractedData) Value() (driver.Value, error) {
	if len(e) == 0 {
		return nil, nil
	}
	return []byte(e), nil
}

// Document represents a financial document uploaded by a tenant.
// It stores metadata, the S3 storage key, processing status, and any extracted JSON data.
type Document struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	TenantID      uint           `gorm:"index;not null" json:"tenant_id"` // multi‑tenant isolation
	OwnerID       uint           `gorm:"index;not null" json:"owner_id"`  // user who performed the upload
	FileName      string         `gorm:"type:varchar(255);not null" json:"file_name"`
	ContentType   string         `gorm:"type:varchar(100);not null" json:"content_type"`
	Size          int64          `gorm:"not null" json:"size"` // size in bytes
	StorageKey    string         `gorm:"type:varchar(512);uniqueIndex;not null" json:"storage_key"` // S3 object key
	Status        DocumentStatus `gorm:"type:varchar(20);index;not null;default:'uploaded'" json:"status"`
	ExtractedData ExtractedData  `gorm:"type:jsonb" json:"extracted_data,omitempty"`
	ErrorMessage  string         `gorm:"type:text" json:"error_message,omitempty"` // populated when Status == failed
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides the default table name used by GORM.
func (Document) TableName() string {
	return "documents"
}

// SetStatus updates the document's processing status.
func (d *Document) SetStatus(status DocumentStatus) {
	d.Status = status
}

// SetExtractedData marshals a Go value into the ExtractedData JSON column.
func (d *Document) SetExtractedData(data interface{}) error {
	if data == nil {
		d.ExtractedData = nil
		return nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal extracted data: %w", err)
	}
	d.ExtractedData = ExtractedData(b)
	return nil
}

// GetExtractedData unmarshals the stored JSON into the provided destination.
// If no data is present, the destination is left untouched and nil is returned.
func (d *Document) GetExtractedData(dest interface{}) error {
	if len(d.ExtractedData) == 0 {
		return nil
	}
	if err := json.Unmarshal(d.ExtractedData, dest); err != nil {
		return fmt.Errorf("failed to unmarshal extracted data: %w", err)
	}
	return nil
}

// BeforeCreate is a GORM hook that ensures sensible defaults before persisting.
func (d *Document) BeforeCreate(tx *gorm.DB) (err error) {
	if d.Status == "" {
		d.Status = DocumentStatusUploaded
	}
	return nil
}