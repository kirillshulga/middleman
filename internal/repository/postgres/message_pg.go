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
			room_id,
			source_endpoint_id,
			source_external_message_id,
			sender,
			text,
			created_at,
			received_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING global_seq
	`,
		msg.ID,
		msg.RoomID,
		msg.SourceEndpointID,
		msg.SourceExternalMessageID,
		msg.Sender,
		msg.Text,
		msg.CreatedAt,
		msg.ReceivedAt,
	).Scan(&msg.GlobalSeq)

	return err
}

func (r *MessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT m.id,
		       m.global_seq,
		       m.room_id,
		       m.source_endpoint_id,
		       e.platform,
		       m.source_external_message_id,
		       m.sender,
		       m.text,
		       m.created_at,
		       m.received_at
		FROM messages m
		JOIN endpoints e ON e.id = m.source_endpoint_id
		WHERE m.id = $1
	`, id)

	var msg domain.Message

	err := row.Scan(
		&msg.ID,
		&msg.GlobalSeq,
		&msg.RoomID,
		&msg.SourceEndpointID,
		&msg.SourcePlatform,
		&msg.SourceExternalMessageID,
		&msg.Sender,
		&msg.Text,
		&msg.CreatedAt,
		&msg.ReceivedAt,
	)

	if err != nil {
		return nil, err
	}

	return &msg, nil
}
