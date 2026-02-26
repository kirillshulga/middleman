package repository

import (
	"context"

	"github.com/google/uuid"

	"middleman/internal/domain"
)

type DeliveryRepository interface {
	CreateBatch(ctx context.Context, tx Tx, deliveries []domain.Delivery) error
	PickPending(ctx context.Context, limit int) ([]domain.Delivery, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DeliveryStatus, lastErr *string) error
}
