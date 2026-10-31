package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"finSight/internal/config"
	"finSight/internal/queue"
	"finSight/internal/service"
	"finSight/internal/util/logger"
	"finSight/internal/queue/task"
)

func main() {
	// Create root context that cancels on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		// Configuration errors are fatal – we cannot continue.
		zap.S().Fatalw("failed to load configuration", "error", err)
	}

	// Initialise structured logger.
	log := logger.New(cfg.LogLevel)

	// Initialise services.
	docSvc, err := service.NewDocumentService(cfg)
	if err != nil {
		log.Fatal("failed to initialise document service", zap.Error(err))
	}
	ocrSvc, err := service.NewOCRService(cfg)
	if err != nil {
		log.Fatal("failed to initialise OCR service", zap.Error(err))
	}
	llmSvc, err := service.NewLLMService(cfg)
	if err != nil {
		log.Fatal("failed to initialise LLM service", zap.Error(err))
	}
	webhookSvc, err := service.NewWebhookService(cfg)
	if err != nil {
		log.Fatal("failed to initialise webhook service", zap.Error(err))
	}

	// Initialise task queue client (Google Cloud Tasks or BullMQ fallback).
	var q queue.Queue
	switch cfg.QueueProvider {
	case "google":
		q, err = queue.NewGoogleTasksClient(ctx, cfg)
		if err != nil {
			log.Fatal("failed to create Google Tasks client", zap.Error(err))
		}
	default:
		q, err = queue.NewBullMQClient(cfg)
		if err != nil {
			log.Fatal("failed to create BullMQ client", zap.Error(err))
		}
	}
	log.Infow("worker started", "queueProvider", cfg.QueueProvider)

	// Main processing loop.
	for {
		// Graceful shutdown.
		select {
		case <-ctx.Done():
			log.Info("shutdown signal received – exiting worker")
			return
		default:
			// continue
		}

		// Pull a single task from the queue.
		t, err := q.Pull(ctx)
		if err != nil {
			// Transient errors – log and back‑off before retrying.
			log.Error("failed to pull task from queue", zap.Error(err))
			time.Sleep(5 * time.Second)
			continue
		}
		if t == nil {
			// No work available – short sleep to avoid hot loop.
			time.Sleep(1 * time.Second)
			continue
		}

		// Process the task.
		if err := handleTask(ctx, t, docSvc, ocrSvc, llmSvc, webhookSvc, log); err != nil {
			log.Error("task processing failed", zap.String("taskID", t.ID), zap.Error(err))
			// Nack with a delay for retry (implementation‑specific; ignore error on nack).
			_ = q.Nack(ctx, t.ID, 1*time.Minute)
			continue
		}

		// Acknowledge successful processing.
		if err := q.Ack(ctx, t.ID); err != nil {
			log.Error("failed to acknowledge task", zap.String("taskID", t.ID), zap.Error(err))
			// If ack fails we let the loop continue; the task may be redelivered.
		}
	}
}

// handleTask performs the full processing pipeline for a single task.
func handleTask(
	ctx context.Context,
	t *task.Task,
	docSvc *service.DocumentService,
	ocrSvc *service.OCRService,
	llmSvc *service.LLMService,
	webhookSvc *service.WebhookService,
	log *zap.Logger,
) error {
	// Decode task payload – expected shape: { "document_id": "<uuid>" }
	var payload struct {
		DocumentID string `json:"document_id"`
	}
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return err
	}
	if payload.DocumentID == "" {
		return errInvalidPayload
	}

	// Fetch document metadata.
	doc, err := docSvc.GetByID(ctx, payload.DocumentID)
	if err != nil {
		return err
	}

	// Run OCR on the stored file.
	rawText, err := ocrSvc.ExtractText(ctx, doc.StorageKey)
	if err != nil {
		return err
	}
	log.Debug("OCR completed", zap.String("documentID", doc.ID), zap.Int("textLength", len(rawText)))

	// Enrich data using the LLM.
	extracted, err := llmSvc.Parse(ctx, rawText)
	if err != nil {
		return err
	}
	log.Debug("LLM parsing completed", zap.String("documentID", doc.ID), zap.Any("extractedKeys", extracted))

	// Persist extraction results and update document status.
	if err := docSvc.UpdateExtraction(ctx, doc.ID, extracted, rawText); err != nil {
		return err
	}
	log.Info("document extraction stored", zap.String("documentID", doc.ID))

	// Fire webhook notifications (non‑blocking – errors are logged but do not cause retry).
	if err := webhookSvc.DispatchDocumentProcessed(ctx, doc.ID, extracted); err != nil {
		log.Warn("webhook dispatch failed", zap.String("documentID", doc.ID), zap.Error(err))
	}
	return nil
}

// errInvalidPayload is returned when a task payload cannot be interpreted.
var errInvalidPayload = &taskError{"invalid task payload"}

type taskError struct {
	msg string
}

func (e *taskError) Error() string { return e.msg }