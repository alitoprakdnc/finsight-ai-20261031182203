package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"finSight/internal/models"
	"finSight/internal/repository"
	"finSight/internal/service/processing"
)

// DocumentService encapsulates business logic around document lifecycle.
type DocumentService struct {
	docRepo       repository.DocumentRepository
	auditRepo     repository.AuditRepository
	processor     processing.Processor
	logger        *zap.Logger
}

// NewDocumentService creates a new DocumentService instance.
func NewDocumentService(
	docRepo repository.DocumentRepository,
	auditRepo repository.AuditRepository,
	processor processing.Processor,
	logger *zap.Logger,
) *DocumentService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DocumentService{
		docRepo:   docRepo,
		auditRepo: auditRepo,
		processor: processor,
		logger:    logger,
	}
}

// CreateDocument registers a new document metadata entry and records an audit event.
// It does NOT upload the file itself – the caller must obtain a pre‑signed URL
// (handled elsewhere) using the returned document.StorageKey.
func (s *DocumentService) CreateDocument(
	ctx context.Context,
	tenantID string,
	ownerID string,
	filename string,
	contentType string,
	size int64,
) (*models.Document, error) {

	docID := uuid.New()
	storageKey := fmt.Sprintf("%s/documents/%s/%s", tenantID, docID.String(), filename)

	doc := &models.Document{
		ID:          docID,
		TenantID:    tenantID,
		OwnerID:     ownerID,
		Filename:    filename,
		ContentType: contentType,
		Size:        size,
		Status:      models.DocumentStatusUploaded,
		StorageKey:  storageKey,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Persist the document record.
	if err := s.docRepo.Create(ctx, doc); err != nil {
		s.logger.Error("failed to create document record", zap.Error(err), zap.String("docID", docID.String()))
		return nil, fmt.Errorf("could not store document metadata: %w", err)
	}

	// Record audit log.
	audit := &models.AuditLog{
		ID:         uuid.New(),
		TenantID:   tenantID,
		UserID:     ownerID,
		Action:     "document_created",
		EntityID:   docID,
		EntityType: "document",
		Timestamp:  time.Now().UTC(),
		Details:    fmt.Sprintf("filename=%s size=%d content_type=%s", filename, size, contentType),
	}
	if err := s.auditRepo.Create(ctx, audit); err != nil {
		// Non‑fatal – log and continue.
		s.logger.Warn("failed to write audit log for document creation", zap.Error(err), zap.String("docID", docID.String()))
	}

	return doc, nil
}

// GetDocument retrieves a document by its ID, ensuring the tenant scope.
func (s *DocumentService) GetDocument(ctx context.Context, tenantID string, docID uuid.UUID) (*models.Document, error) {
	doc, err := s.docRepo.GetByID(ctx, docID)
	if err != nil {
		s.logger.Error("failed to fetch document", zap.Error(err), zap.String("docID", docID.String()))
		return nil, fmt.Errorf("could not retrieve document: %w", err)
	}
	if doc.TenantID != tenantID {
		return nil, fmt.Errorf("document does not belong to tenant")
	}
	return doc, nil
}

// UpdateStatus transitions a document to a new status and records the change in the audit log.
func (s *DocumentService) UpdateStatus(
	ctx context.Context,
	tenantID string,
	userID string,
	docID uuid.UUID,
	newStatus models.DocumentStatus,
) (*models.Document, error) {

	// Load current document.
	doc, err := s.docRepo.GetByID(ctx, docID)
	if err != nil {
		s.logger.Error("failed to load document for status update", zap.Error(err), zap.String("docID", docID.String()))
		return nil, fmt.Errorf("could not load document: %w", err)
	}
	if doc.TenantID != tenantID {
		return nil, fmt.Errorf("document does not belong to tenant")
	}

	oldStatus := doc.Status
	if oldStatus == newStatus {
		// No change required.
		return doc, nil
	}

	// Persist the new status.
	if err := s.docRepo.UpdateStatus(ctx, docID, newStatus); err != nil {
		s.logger.Error("failed to update document status", zap.Error(err), zap.String("docID", docID.String()))
		return nil, fmt.Errorf("could not update status: %w", err)
	}
	doc.Status = newStatus
	doc.UpdatedAt = time.Now().UTC()

	// Audit entry.
	audit := &models.AuditLog{
		ID:         uuid.New(),
		TenantID:   tenantID,
		UserID:     userID,
		Action:     "document_status_updated",
		EntityID:   docID,
		EntityType: "document",
		Timestamp:  time.Now().UTC(),
		Details:    fmt.Sprintf("from=%s to=%s", oldStatus, newStatus),
	}
	if err := s.auditRepo.Create(ctx, audit); err != nil {
		s.logger.Warn("failed to write audit log for status change", zap.Error(err), zap.String("docID", docID.String()))
	}

	return doc, nil
}

// TriggerProcessing enqueues a document for OCR + LLM processing.
// It also moves the document status to `processing` and records an audit event.
func (s *DocumentService) TriggerProcessing(
	ctx context.Context,
	tenantID string,
	userID string,
	docID uuid.UUID,
) error {

	// Ensure the document exists and belongs to the tenant.
	doc, err := s.docRepo.GetByID(ctx, docID)
	if err != nil {
		s.logger.Error("failed to fetch document for processing", zap.Error(err), zap.String("docID", docID.String()))
		return fmt.Errorf("could not fetch document: %w", err)
	}
	if doc.TenantID != tenantID {
		return fmt.Errorf("document does not belong to tenant")
	}
	if doc.Status != models.DocumentStatusUploaded && doc.Status != models.DocumentStatusFailed {
		return fmt.Errorf("document is not in a state that can be processed (current: %s)", doc.Status)
	}

	// Update status to processing.
	if err := s.docRepo.UpdateStatus(ctx, docID, models.DocumentStatusProcessing); err != nil {
		s.logger.Error("failed to set document status to processing", zap.Error(err), zap.String("docID", docID.String()))
		return fmt.Errorf("could not update status: %w", err)
	}
	doc.Status = models.DocumentStatusProcessing
	doc.UpdatedAt = time.Now().UTC()

	// Enqueue the background task.
	if err := s.processor.EnqueueDocument(ctx, docID.String()); err != nil {
		// Roll back status change to avoid orphaned state.
		rollbackErr := s.docRepo.UpdateStatus(ctx, docID, models.DocumentStatusUploaded)
		if rollbackErr != nil {
			s.logger.Error("failed to rollback document status after enqueue failure", zap.Error(rollbackErr), zap.String("docID", docID.String()))
		}
		s.logger.Error("failed to enqueue document processing task", zap.Error(err), zap.String("docID", docID.String()))
		return fmt.Errorf("could not enqueue processing task: %w", err)
	}

	// Audit log for the trigger.
	audit := &models.AuditLog{
		ID:         uuid.New(),
		TenantID:   tenantID,
		UserID:     userID,
		Action:     "document_processing_queued",
		EntityID:   docID,
		EntityType: "document",
		Timestamp:  time.Now().UTC(),
		Details:    "task enqueued",
	}
	if err := s.auditRepo.Create(ctx, audit); err != nil {
		s.logger.Warn("failed to write audit log for processing trigger", zap.Error(err), zap.String("docID", docID.String()))
	}

	return nil
}