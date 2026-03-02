package postgres

import (
	"context"
	"time"

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
			INSERT INTO deliveries (
				id,
				message_id,
				target_endpoint_id,
				status,
				attempts,
				next_retry_at,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
			d.ID,
			d.MessageID,
			d.TargetEndpointID,
			d.Status,
			d.Attempts,
			d.NextRetryAt,
			d.CreatedAt,
			d.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *DeliveryRepository) ClaimPending(ctx context.Context, limit int) ([]domain.Delivery, error) {
	rows, err := r.pool.Query(ctx, `
		WITH picked AS (
			SELECT id
			FROM deliveries
			WHERE status = 'pending'
			  AND next_retry_at <= NOW()
			ORDER BY next_retry_at, created_at
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		UPDATE deliveries d
		SET status = 'processing',
		    updated_at = NOW()
		FROM picked
		WHERE d.id = picked.id
		RETURNING d.id, d.message_id, d.target_endpoint_id, d.status, d.attempts, d.next_retry_at, d.last_error, d.created_at, d.updated_at
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
			&d.TargetEndpointID,
			&d.Status,
			&d.Attempts,
			&d.NextRetryAt,
			&d.LastError,
			&d.CreatedAt,
			&d.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		result = append(result, d)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *DeliveryRepository) MarkSent(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE deliveries
		SET status = 'sent',
		    last_error = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`, id)

	return err
}

func (r *DeliveryRepository) MarkRetry(ctx context.Context, id uuid.UUID, lastErr string, nextRetryAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE deliveries
		SET status = 'pending',
		    last_error = $1,
		    attempts = attempts + 1,
		    next_retry_at = $2,
		    updated_at = NOW()
		WHERE id = $3
	`, lastErr, nextRetryAt, id)

	return err
}

func (r *DeliveryRepository) MarkFailed(ctx context.Context, id uuid.UUID, lastErr string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE deliveries
		SET status = 'failed',
		    last_error = $1,
		    attempts = attempts + 1,
		    updated_at = NOW()
		WHERE id = $2
	`, lastErr, id)

	return err
}
