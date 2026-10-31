package service

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/yourorg/finSight/internal/config"
	"github.com/yourorg/finSight/internal/models"
	"github.com/yourorg/finSight/internal/queue"
)

// ProcessingService is responsible for pushing document‑processing jobs onto the
// configured task queue (Google Cloud Tasks or BullMQ). It abstracts the queue
// implementation so callers only need to provide a document model.
type ProcessingService struct {
	cfg          *config.Config
	logger       *zap.Logger
	googleClient *queue.GoogleTaskClient
	bullClient   *queue.BullMQClient
}

// NewProcessingService creates a new ProcessingService instance. It initialises
// the appropriate queue client based on the configuration. If the configured
// provider cannot be initialised, an error is returned and the service should
// not be used.
func NewProcessingService(cfg *config.Config, logger *zap.Logger) (*ProcessingService, error) {
	ps := &ProcessingService{
		cfg:    cfg,
		logger: logger,
	}

	switch cfg.QueueProvider {
	case "google":
		gtc, err := queue.NewGoogleTaskClient(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialise Google Cloud Tasks client: %w", err)
		}
		ps.googleClient = gtc

	case "bullmq":
		bc, err := queue.NewBullMQClient(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialise BullMQ client: %w", err)
		}
		ps.bullClient = bc

	default:
		// Default to BullMQ – it works locally without GCP credentials.
		bc, err := queue.NewBullMQClient(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialise default BullMQ client: %w", err)
		}
		ps.bullClient = bc
	}

	return ps, nil
}

// EnqueueDocumentProcessing creates a task payload for the given document and
// pushes it onto the configured queue. The payload contains the minimal data
// required by the worker to locate and process the document.
func (ps *ProcessingService) EnqueueDocumentProcessing(ctx context.Context, doc *models.Document) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}
	if doc.ID == "" {
		return fmt.Errorf("document ID is empty")
	}

	// Build the task payload – keep it small and JSON‑serialisable.
	payload := queue.TaskMessage{
		DocumentID: doc.ID,
		TenantID:   doc.TenantID,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task payload: %w", err)
	}

	// Prefer Google Cloud Tasks when it is configured; fall back to BullMQ
	// on any error (or when the provider is BullMQ from the start).
	if ps.googleClient != nil {
		if err := ps.googleClient.CreateTask(ctx, data); err != nil {
			ps.logger.Error("Google Cloud Tasks enqueue failed, attempting BullMQ fallback",
				zap.Error(err), zap.String("document_id", doc.ID))

			if ps.bullClient != nil {
				if err2 := ps.bullClient.EnqueueTask(ctx, data); err2 != nil {
					return fmt.Errorf("google tasks error: %w; bullmq fallback error: %w", err, err2)
				}
				ps.logger.Info("Task enqueued via BullMQ fallback", zap.String("document_id", doc.ID))
				return nil
			}
			return fmt.Errorf("google tasks enqueue error: %w", err)
		}
		ps.logger.Info("Task enqueued to Google Cloud Tasks", zap.String("document_id", doc.ID))
		return nil
	}

	if ps.bullClient != nil {
		if err := ps.bullClient.EnqueueTask(ctx, data); err != nil {
			ps.logger.Error("BullMQ enqueue failed", zap.Error(err), zap.String("document_id", doc.ID))
			return fmt.Errorf("bullmq enqueue error: %w", err)
		}
		ps.logger.Info("Task enqueued to BullMQ", zap.String("document_id", doc.ID))
		return nil
	}

	return fmt.Errorf("no queue client configured")
}