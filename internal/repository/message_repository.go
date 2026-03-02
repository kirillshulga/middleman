package repository

import (
	"context"

	"github.com/google/uuid"

	"middleman/internal/domain"
)

type MessageRepository interface {
	Create(ctx context.Context, tx Tx, msg *domain.Message) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error)
}
