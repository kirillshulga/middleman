package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"middleman/internal/domain"
)

type DeliveryQueueStats struct {
	FailedCount         int64
	BacklogPendingCount int64
	RetrySpikeCount     int64
}

type DeliveryRepository interface {
	CreateBatch(ctx context.Context, tx Tx, deliveries []domain.Delivery) error
	ClaimPending(ctx context.Context, limit int) ([]domain.Delivery, error)
	MarkSent(ctx context.Context, id uuid.UUID) error
	MarkRetry(ctx context.Context, id uuid.UUID, lastErr string, nextRetryAt time.Time) error
	MarkFailed(ctx context.Context, id uuid.UUID, lastErr string) error
	GetQueueStats(ctx context.Context, retrySince time.Time) (DeliveryQueueStats, error)
}
