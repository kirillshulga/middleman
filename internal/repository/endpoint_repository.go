package repository

import (
	"context"

	"github.com/google/uuid"

	"middleman/internal/domain"
)

type EndpointRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Endpoint, error)
	GetByPlatformChatID(ctx context.Context, platform domain.Platform, externalChatID string) (*domain.Endpoint, error)
	ListActiveByRoom(ctx context.Context, roomID uuid.UUID) ([]domain.Endpoint, error)
}
