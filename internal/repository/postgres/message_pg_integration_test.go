package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"middleman/internal/domain"
)

func TestMessageRepository_Integration_CreateAndGetByID(t *testing.T) {
	pool := setupIntegrationDB(t)
	msgRepo := NewMessageRepository(pool)
	txMgr := NewTxManager(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	roomID := uuid.New()
	endpointID := uuid.New()
	messageID := uuid.New()

	_, err := pool.Exec(ctx, `
		INSERT INTO rooms (id, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, roomID, "room-a")
	if err != nil {
		t.Fatalf("insert room error = %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO endpoints (id, room_id, platform, external_chat_id, status, created_at, updated_at)
		VALUES ($1, $2, 'telegram', 'tg-1', 'active', NOW(), NOW())
	`, endpointID, roomID)
	if err != nil {
		t.Fatalf("insert endpoint error = %v", err)
	}

	tx, err := txMgr.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	msg := &domain.Message{
		ID:                      messageID,
		RoomID:                  roomID,
		SourceEndpointID:        endpointID,
		SourceExternalMessageID: "msg-1",
		Sender:                  "alice",
		Text:                    "hello",
		CreatedAt:               time.Now().Add(-time.Minute).UTC().Truncate(time.Second),
		ReceivedAt:              time.Now().UTC().Truncate(time.Second),
	}

	if err := msgRepo.Create(ctx, tx, msg); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	got, err := msgRepo.GetByID(ctx, messageID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.ID != messageID {
		t.Fatalf("unexpected message id: got %s want %s", got.ID, messageID)
	}
	if got.SourcePlatform != domain.PlatformTelegram {
		t.Fatalf("unexpected source platform: got %s", got.SourcePlatform)
	}
	if got.SourceExternalMessageID != "msg-1" {
		t.Fatalf("unexpected source external message id: got %s", got.SourceExternalMessageID)
	}
	if got.GlobalSeq < 1 {
		t.Fatalf("unexpected global seq: %d", got.GlobalSeq)
	}
}
