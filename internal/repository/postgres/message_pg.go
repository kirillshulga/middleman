package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"middleman/internal/domain"
	"middleman/internal/repository"
)

type MessageRepository struct {
	pool *pgxpool.Pool
}

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

func (r *MessageRepository) Create(
	ctx context.Context,
	tx repository.Tx,
	msg *domain.Message,
) error {

	pgTx := tx.(*Tx)

	err := pgTx.tx.QueryRow(ctx, `
		INSERT INTO messages (
			id,
			source_platform,
			source_external_id,
			sender,
			text,
			created_at,
			received_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING global_seq
	`,
		msg.ID,
		msg.SourcePlatform,
		msg.SourceExternalID,
		msg.Sender,
		msg.Text,
		msg.CreatedAt,
		msg.ReceivedAt,
	).Scan(&msg.GlobalSeq)

	return err
}

func (r *MessageRepository) ExistsByExternalID(ctx context.Context, platform domain.Platform, externalID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM messages
			WHERE source_platform = $1 AND source_external_id = $2
		)
	`, platform, externalID).Scan(&exists)

	return exists, err
}

func (r *MessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, global_seq, source_platform, source_external_id, sender, text, created_at
		FROM messages
		WHERE id = $1
	`, id)

	var msg domain.Message

	err := row.Scan(
		&msg.ID,
		&msg.GlobalSeq,
		&msg.SourcePlatform,
		&msg.SourceExternalID,
		&msg.Sender,
		&msg.Text,
		&msg.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &msg, nil
}
