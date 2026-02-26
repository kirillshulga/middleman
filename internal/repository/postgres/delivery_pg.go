package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"middleman/internal/domain"
	"middleman/internal/repository"
)

type DeliveryRepository struct {
	pool *pgxpool.Pool
}

func NewDeliveryRepository(pool *pgxpool.Pool) *DeliveryRepository {
	return &DeliveryRepository{pool: pool}
}

func (r *DeliveryRepository) CreateBatch(ctx context.Context, tx repository.Tx, deliveries []domain.Delivery) error {
	pgTx := tx.(*Tx)

	for _, d := range deliveries {
		_, err := pgTx.tx.Exec(ctx, `
			INSERT INTO deliveries (id, message_id, platform, status, attempts, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
			d.ID,
			d.MessageID,
			d.Platform,
			d.Status,
			d.Attempts,
			d.CreatedAt,
			d.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *DeliveryRepository) PickPending(ctx context.Context, limit int) ([]domain.Delivery, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, message_id, platform, status, attempts, created_at, updated_at
		FROM deliveries
		WHERE status = 'pending'
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT $1
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Delivery

	for rows.Next() {
		var d domain.Delivery
		err := rows.Scan(
			&d.ID,
			&d.MessageID,
			&d.Platform,
			&d.Status,
			&d.Attempts,
			&d.CreatedAt,
			&d.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		result = append(result, d)
	}

	return result, nil
}

func (r *DeliveryRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DeliveryStatus, lastErr *string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE deliveries
		SET status = $1,
		    last_error = $2,
		    attempts = attempts + 1,
		    updated_at = NOW()
		WHERE id = $3
	`, status, lastErr, id)

	return err
}
