package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/yourorg/finSight/internal/models"
)

// AuditFilter defines optional criteria for querying audit logs.
type AuditFilter struct {
	TenantID  string    // restrict to a specific tenant (multi‑tenant isolation)
	UserID    string    // actions performed by a specific user
	Action    string    // e.g. "document.upload", "webhook.trigger"
	StartTime time.Time // inclusive lower bound for timestamp
	EndTime   time.Time // inclusive upper bound for timestamp
	Limit     int       // max rows to return (defaults to 100)
	Offset    int       // pagination offset
}

// AuditRepository provides write‑only and read‑only access to the immutable audit_logs table.
type AuditRepository struct {
	db *gorm.DB
}

// NewAuditRepository creates a new instance of AuditRepository.
// The caller is responsible for providing a *gorm.DB that is already configured
// (dialect, connection pool, logger, etc.).
func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create inserts a new audit log entry.
// The entry must contain all required fields; the repository does not modify it.
// Returns an error if the insert fails.
func (r *AuditRepository) Create(ctx context.Context, entry *models.AuditLog) error {
	if entry == nil {
		return gorm.ErrInvalidData
	}
	return r.db.WithContext(ctx).Create(entry).Error
}

// List returns audit log entries that match the supplied filter.
// The result set is ordered by timestamp descending (most recent first).
// If filter.Limit is zero, a sane default of 100 rows is applied.
func (r *AuditRepository) List(ctx context.Context, filter AuditFilter) ([]models.AuditLog, error) {
	var logs []models.AuditLog

	q := r.db.WithContext(ctx).Model(&models.AuditLog{})

	if filter.TenantID != "" {
		q = q.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.UserID != "" {
		q = q.Where("user_id = ?", filter.UserID)
	}
	if filter.Action != "" {
		q = q.Where("action = ?", filter.Action)
	}
	if !filter.StartTime.IsZero() {
		q = q.Where("created_at >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		q = q.Where("created_at <= ?", filter.EndTime)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	q = q.Limit(limit)

	if filter.Offset > 0 {
		q = q.Offset(filter.Offset)
	}

	err := q.Order("created_at DESC").Find(&logs).Error
	return logs, err
}

// Count returns the total number of audit log entries that satisfy the filter.
// Useful for pagination UI where the total count is displayed.
func (r *AuditRepository) Count(ctx context.Context, filter AuditFilter) (int64, error) {
	var cnt int64

	q := r.db.WithContext(ctx).Model(&models.AuditLog{})

	if filter.TenantID != "" {
		q = q.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.UserID != "" {
		q = q.Where("user_id = ?", filter.UserID)
	}
	if filter.Action != "" {
		q = q.Where("action = ?", filter.Action)
	}
	if !filter.StartTime.IsZero() {
		q = q.Where("created_at >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		q = q.Where("created_at <= ?", filter.EndTime)
	}

	err := q.Count(&cnt).Error
	return cnt, err
}