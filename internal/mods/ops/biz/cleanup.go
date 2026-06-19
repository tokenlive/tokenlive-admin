package biz

import (
	"context"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	cleanupInterval = 6 * time.Hour
	retentionDays   = 7
)

// CleanupTask periodically deletes event logs older than the retention period.
type CleanupTask struct {
	DB *gorm.DB
}

// Start begins the cleanup goroutine.
func (t *CleanupTask) Start(ctx context.Context) {
	go t.run(ctx)
}

func (t *CleanupTask) run(ctx context.Context) {
	logging.Context(ctx).Info("event cleanup task started",
		zap.Duration("interval", cleanupInterval),
		zap.Int("retention_days", retentionDays),
	)

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	// Run once at startup
	t.cleanup(ctx)

	for {
		select {
		case <-ctx.Done():
			logging.Context(ctx).Info("event cleanup task stopped")
			return
		case <-ticker.C:
			t.cleanup(ctx)
		}
	}
}

func (t *CleanupTask) cleanup(ctx context.Context) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result := t.DB.WithContext(ctx).
		Where("event_time < ?", cutoff).
		Delete(&schema.EventLog{})
	if result.Error != nil {
		logging.Context(ctx).Error("event_log cleanup failed", zap.Error(result.Error))
	} else if result.RowsAffected > 0 {
		logging.Context(ctx).Info("event_log cleanup completed",
			zap.Int64("deleted", result.RowsAffected),
			zap.Time("cutoff", cutoff),
		)
	}
}
