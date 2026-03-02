package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"middleman/internal/domain"
)

type EndpointRepository struct {
	pool *pgxpool.Pool
}

func NewEndpointRepository(pool *pgxpool.Pool) *EndpointRepository {
	return &EndpointRepository{pool: pool}
}

func (r *EndpointRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Endpoint, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, room_id, platform, external_chat_id, status, created_at, updated_at
		FROM endpoints
		WHERE id = $1
	`, id)

	var endpoint domain.Endpoint
	if err := row.Scan(
		&endpoint.ID,
		&endpoint.RoomID,
		&endpoint.Platform,
		&endpoint.ExternalChatID,
		&endpoint.Status,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &endpoint, nil
}

func (r *EndpointRepository) GetByPlatformChatID(ctx context.Context, platform domain.Platform, externalChatID string) (*domain.Endpoint, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, room_id, platform, external_chat_id, status, created_at, updated_at
		FROM endpoints
		WHERE platform = $1
		  AND external_chat_id = $2
		  AND status = 'active'
	`, platform, externalChatID)

	var endpoint domain.Endpoint
	if err := row.Scan(
		&endpoint.ID,
		&endpoint.RoomID,
		&endpoint.Platform,
		&endpoint.ExternalChatID,
		&endpoint.Status,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &endpoint, nil
}

func (r *EndpointRepository) ListActiveByRoom(ctx context.Context, roomID uuid.UUID) ([]domain.Endpoint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, room_id, platform, external_chat_id, status, created_at, updated_at
		FROM endpoints
		WHERE room_id = $1
		  AND status = 'active'
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.Endpoint, 0)
	for rows.Next() {
		var endpoint domain.Endpoint
		if err := rows.Scan(
			&endpoint.ID,
			&endpoint.RoomID,
			&endpoint.Platform,
			&endpoint.ExternalChatID,
			&endpoint.Status,
			&endpoint.CreatedAt,
			&endpoint.UpdatedAt,
		); err != nil {
			return nil, err
		}

		result = append(result, endpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
