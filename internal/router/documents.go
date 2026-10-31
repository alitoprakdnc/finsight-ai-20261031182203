package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"finSight/internal/auth"
	"finSight/internal/models"
	"finSight/internal/service"
	"finSight/internal/util/response"
	"finSight/internal/util/validation"
)

// RegisterDocumentRoutes registers all document related endpoints.
func RegisterDocumentRoutes(rg *gin.RouterGroup, docSvc *service.DocumentService, procSvc *service.ProcessingService) {
	docGroup := rg.Group("/documents")
	{
		docGroup.POST("", auth.RequireRoles([]string{"admin", "accountant"}), createDocumentHandler(docSvc, procSvc))
		docGroup.GET("", auth.RequireRoles([]string{"admin", "accountant", "viewer"}), listDocumentsHandler(docSvc))
		docGroup.GET("/:id", auth.RequireRoles([]string{"admin", "accountant", "viewer"}), getDocumentHandler(docSvc))
		docGroup.DELETE("/:id", auth.RequireRoles([]string{"admin"}), deleteDocumentHandler(docSvc))
		docGroup.GET("/:id/status", auth.RequireRoles([]string{"admin", "accountant", "viewer"}), getDocumentStatusHandler(docSvc))
	}
}

// -----------------------------------------------------------------------------
// Request / Response payloads
// -----------------------------------------------------------------------------

type createDocumentRequest struct {
	FileName    string `json:"file_name" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
	Size        int64  `json:"size" binding:"required,gt=0"`
}

type createDocumentResponse struct {
	ID        string    `json:"id"`
	UploadURL string    `json:"upload_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type documentResponse struct {
	ID           string                 `json:"id"`
	FileName     string                 `json:"file_name"`
	ContentType  string                 `json:"content_type"`
	Size         int64                  `json:"size"`
	Status       models.DocumentStatus `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Extracted    map[string]interface{} `json:"extracted,omitempty"`
	DownloadURL  string                 `json:"download_url,omitempty"`
	ProcessedAt  *time.Time             `json:"processed_at,omitempty"`
	FailureReason *string               `json:"failure_reason,omitempty"`
}

type listDocumentsResponse struct {
	Documents []documentResponse `json:"documents"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

// -----------------------------------------------------------------------------
// Handlers
// -----------------------------------------------------------------------------

func createDocumentHandler(docSvc *service.DocumentService, procSvc *service.ProcessingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createDocumentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, validation.ErrInvalidRequest(err))
			return
		}

		claims, err := auth.GetClaims(c)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, err)
			return
		}

		// Create a new document record (status = pending_upload)
		doc := &models.Document{
			ID:          uuid.NewString(),
			TenantID:    claims.TenantID,
			FileName:    req.FileName,
			ContentType: req.ContentType,
			Size:        req.Size,
			Status:      models.StatusPendingUpload,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		if err := docSvc.Create(c.Request.Context(), doc); err != nil {
			response.Error(c, http.StatusInternalServerError, err)
			return
		}

		// Generate a pre‑signed upload URL
		uploadURL, expiresAt, err := docSvc.GenerateUploadURL(c.Request.Context(), doc.ID, req.ContentType)
		if err != nil {
			// Cleanup the document record if we cannot provide an upload URL
			_ = docSvc.Delete(c.Request.Context(), doc.ID, claims.TenantID)
			response.Error(c, http.StatusInternalServerError, err)
			return
		}

		// Enqueue processing task – the worker will wait for the file to appear.
		if err := procSvc.EnqueueDocument(c.Request.Context(), doc.ID); err != nil {
			// Not fatal for the client, but log for ops.
			// The task will be retried by the worker if needed.
			// We still return the upload URL.
		}

		resp := createDocumentResponse{
			ID:        doc.ID,
			UploadURL: uploadURL,
			ExpiresAt: expiresAt,
		}
		response.Success(c, http.StatusCreated, resp)
	}
}

func listDocumentsHandler(docSvc *service.DocumentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := auth.GetClaims(c)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, err)
			return
		}

		// Pagination params
		limit, cursor, err := validation.ParsePaginationParams(c)
		if err != nil {
			response.Error(c, http.StatusBadRequest, err)
			return
		}

		filter := &service.DocumentFilter{
			TenantID: claims.TenantID,
			Status:   c.Query("status"),
		}

		docs, nextCursor, err := docSvc.List(c.Request.Context(), filter, limit, cursor)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err)
			return
		}

		respDocs := make([]documentResponse, len(docs))
		for i, d := range docs {
			respDocs[i] = toDocumentResponse(c.Request.Context(), d)
		}

		resp := listDocumentsResponse{
			Documents: respDocs,
			NextCursor: nextCursor,
		}
		response.Success(c, http.StatusOK, resp)
	}
}

func getDocumentHandler(docSvc *service.DocumentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := auth.GetClaims(c)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, err)
			return
		}
		docID := c.Param("id")
		if docID == "" {
			response.Error(c, http.StatusBadRequest, validation.ErrMissingParameter("id"))
			return
		}

		doc, err := docSvc.GetByID(c.Request.Context(), claims.TenantID, docID)
		if err != nil {
			if err == service.ErrNotFound {
				response.Error(c, http.StatusNotFound, err)
			} else {
				response.Error(c, http.StatusInternalServerError, err)
			}
			return
		}

		resp := toDocumentResponse(c.Request.Context(), doc)
		response.Success(c, http.StatusOK, resp)
	}
}

func deleteDocumentHandler(docSvc *service.DocumentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := auth.GetClaims(c)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, err)
			return
		}
		docID := c.Param("id")
		if docID == "" {
			response.Error(c, http.StatusBadRequest, validation.ErrMissingParameter("id"))
			return
		}

		if err := docSvc.Delete(c.Request.Context(), docID, claims.TenantID); err != nil {
			if err == service.ErrNotFound {
				response.Error(c, http.StatusNotFound, err)
			} else {
				response.Error(c, http.StatusInternalServerError, err)
			}
			return
		}
		response.Success(c, http.StatusNoContent, nil)
	}
}

func getDocumentStatusHandler(docSvc *service.DocumentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := auth.GetClaims(c)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, err)
			return
		}
		docID := c.Param("id")
		if docID == "" {
			response.Error(c, http.StatusBadRequest, validation.ErrMissingParameter("id"))
			return
		}

		status, err := docSvc.GetStatus(c.Request.Context(), claims.TenantID, docID)
		if err != nil {
			if err == service.ErrNotFound {
				response.Error(c, http.StatusNotFound, err)
			} else {
				response.Error(c, http.StatusInternalServerError, err)
			}
			return
		}
		response.Success(c, http.StatusOK, gin.H{
			"id":     docID,
			"status": status,
		})
	}
}

// -----------------------------------------------------------------------------
// Helper conversion
// -----------------------------------------------------------------------------

func toDocumentResponse(ctx context.Context, d *models.Document) documentResponse {
	resp := documentResponse{
		ID:          d.ID,
		FileName:    d.FileName,
		ContentType: d.ContentType,
		Size:        d.Size,
		Status:      d.Status,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}

	if d.ExtractedData != nil {
		resp.Extracted = d.ExtractedData
	}
	if d.ProcessedAt != nil {
		resp.ProcessedAt = d.ProcessedAt
	}
	if d.FailureReason != nil {
		resp.FailureReason = d.FailureReason
	}
	// Generate a short‑lived download URL if the document is ready
	if d.Status == models.StatusProcessed {
		if url, _, err := service.GetDownloadURL(ctx, d.ID); err == nil {
			resp.DownloadURL = url
		}
	}
	return resp
}