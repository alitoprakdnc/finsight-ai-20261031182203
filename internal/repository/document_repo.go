package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"finSight/internal/models"
)

// ListOptions defines optional filters for listing documents.
type ListOptions struct {
	Limit    int
	Offset   int
	Status   string
	Search   string // searches filename or vendor name in extracted data
	OrderBy  string // e.g. "created_at desc"
}

// DocumentRepository defines the contract for document data access.
type DocumentRepository interface {
	// Create inserts a new document record.
	Create(ctx context.Context, doc *models.Document) error
	// GetByID fetches a document belonging to a tenant.
	GetByID(ctx context.Context, tenantID string, documentID uuid.UUID) (*models.Document, error)
	// List returns documents for a tenant applying the supplied options.
	List(ctx context.Context, tenantID string, opts ListOptions) ([]*models.Document, error)
	// UpdateStatus changes the processing status of a document.
	UpdateStatus(ctx context.Context, tenantID string, documentID uuid.UUID, status string) error
	// UpdateExtractedData stores the structured JSON extracted by the LLM.
	UpdateExtractedData(ctx context.Context, tenantID string, documentID uuid.UUID, data json.RawMessage) error
	// Delete removes a document record permanently.
	Delete(ctx context.Context, tenantID string, documentID uuid.UUID) error
	// WithTx returns a repository bound to the supplied transaction.
	WithTx(tx *gorm.DB) DocumentRepository
}

// documentRepo implements DocumentRepository using GORM.
type documentRepo struct {
	db *gorm.DB
}

// NewDocumentRepository creates a new repository instance.
func NewDocumentRepository(db *gorm.DB) DocumentRepository {
	return &documentRepo{db: db}
}

// WithTx returns a new repository that uses the provided transaction.
// This enables callers to compose multiple repository calls inside a single DB transaction.
func (r *documentRepo) WithTx(tx *gorm.DB) DocumentRepository {
	if tx == nil {
		return r
	}
	return &documentRepo{db: tx}
}

// Create inserts a new document record.
func (r *documentRepo) Create(ctx context.Context, doc *models.Document) error {
	if doc == nil {
		return errors.New("document is nil")
	}
	if doc.ID == uuid.Nil {
		doc.ID = uuid.New()
	}
	if err := r.db.WithContext(ctx).Create(doc).Error; err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}
	return nil
}

// GetByID fetches a document belonging to a tenant.
func (r *documentRepo) GetByID(ctx context.Context, tenantID string, documentID uuid.UUID) (*models.Document, error) {
	var doc models.Document
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, documentID).
		First(&doc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("document %s not found: %w", documentID, err)
		}
		return nil, fmt.Errorf("failed to fetch document %s: %w", documentID, err)
	}
	return &doc, nil
}

// List returns documents for a tenant applying the supplied options.
func (r *documentRepo) List(ctx context.Context, tenantID string, opts ListOptions) ([]*models.Document, error) {
	query := r.db.WithContext(ctx).Model(&models.Document{}).Where("tenant_id = ?", tenantID)

	if opts.Status != "" {
		query = query.Where("status = ?", opts.Status)
	}
	if opts.Search != "" {
		// Simple ILIKE search on filename and extracted vendor field (if present)
		searchPattern := fmt.Sprintf("%%%s%%", opts.Search)
		query = query.Where(
			r.db.Where("file_name ILIKE ?", searchPattern).
				Or("extracted_data->>'vendor' ILIKE ?", searchPattern),
		)
	}
	if opts.OrderBy != "" {
		query = query.Order(opts.OrderBy)
	} else {
		query = query.Order("created_at DESC")
	}
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}

	var docs []*models.Document
	if err := query.Find(&docs).Error; err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	return docs, nil
}

// UpdateStatus changes the processing status of a document.
func (r *documentRepo) UpdateStatus(ctx context.Context, tenantID string, documentID uuid.UUID, status string) error {
	result := r.db.WithContext(ctx).
		Model(&models.Document{}).
		Where("tenant_id = ? AND id = ?", tenantID, documentID).
		Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("failed to update status for document %s: %w", documentID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no document found to update status for %s", documentID)
	}
	return nil
}

// UpdateExtractedData stores the structured JSON extracted by the LLM.
func (r *documentRepo) UpdateExtractedData(ctx context.Context, tenantID string, documentID uuid.UUID, data json.RawMessage) error {
	if len(data) == 0 {
		return errors.New("extracted data payload is empty")
	}
	result := r.db.WithContext(ctx).
		Model(&models.Document{}).
		Where("tenant_id = ? AND id = ?", tenantID, documentID).
		Update("extracted_data", data)
	if result.Error != nil {
		return fmt.Errorf("failed to update extracted data for document %s: %w", documentID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no document found to update extracted data for %s", documentID)
	}
	return nil
}

// Delete removes a document record permanently.
func (r *documentRepo) Delete(ctx context.Context, tenantID string, documentID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, documentID).
		Delete(&models.Document{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete document %s: %w", documentID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no document found to delete for %s", documentID)
	}
	return nil
}