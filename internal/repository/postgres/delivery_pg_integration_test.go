package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"middleman/internal/domain"
)

func TestDeliveryRepository_Integration_ClaimAndStateTransitions(t *testing.T) {
	pool := setupIntegrationDB(t)
	repo := NewDeliveryRepository(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	roomID := uuid.New()
	sourceEndpointID := uuid.New()
	targetEndpointID := uuid.New()
	messageID := uuid.New()
	deliveryID := uuid.New()

	_, err := pool.Exec(ctx, `
		INSERT INTO rooms (id, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, roomID, "room-a")
	if err != nil {
		t.Fatalf("insert room error = %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO endpoints (id, room_id, platform, external_chat_id, status, created_at, updated_at)
		VALUES
		  ($1, $3, 'telegram', 'tg-1', 'active', NOW(), NOW()),
		  ($2, $3, 'slack', 'slack-1', 'active', NOW(), NOW())
	`, sourceEndpointID, targetEndpointID, roomID)
	if err != nil {
		t.Fatalf("insert endpoints error = %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO messages (
			id, room_id, source_endpoint_id, source_external_message_id, sender, text, created_at, received_at
		)
		VALUES ($1, $2, $3, 'msg-1', 'alice', 'hello', NOW(), NOW())
	`, messageID, roomID, sourceEndpointID)
	if err != nil {
		t.Fatalf("insert message error = %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO deliveries (
			id, message_id, target_endpoint_id, status, attempts, next_retry_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, 'pending', 0, NOW() - INTERVAL '1 minute', NOW(), NOW())
	`, deliveryID, messageID, targetEndpointID)
	if err != nil {
		t.Fatalf("insert delivery error = %v", err)
	}

	claimed, err := repo.ClaimPending(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimPending() error = %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("expected one claimed delivery, got %d", len(claimed))
	}
	if claimed[0].ID != deliveryID {
		t.Fatalf("unexpected claimed delivery id: got %s want %s", claimed[0].ID, deliveryID)
	}
	if claimed[0].Status != domain.DeliveryProcessing {
		t.Fatalf("expected processing status after claim, got %s", claimed[0].Status)
	}

	nextRetry := time.Now().Add(2 * time.Minute)
	if err := repo.MarkRetry(ctx, deliveryID, "temporary error", nextRetry); err != nil {
		t.Fatalf("MarkRetry() error = %v", err)
	}

	stats, err := repo.GetQueueStats(ctx, time.Now().Add(-10*time.Minute))
	if err != nil {
		t.Fatalf("GetQueueStats() error = %v", err)
	}
	if stats.BacklogPendingCount < 1 {
		t.Fatalf("expected backlog >= 1, got %d", stats.BacklogPendingCount)
	}
	if stats.RetrySpikeCount < 1 {
		t.Fatalf("expected retry spike >= 1, got %d", stats.RetrySpikeCount)
	}

	var status string
	var attempts int
	var lastErr sql.NullString
	if err := pool.QueryRow(ctx, `
		SELECT status, attempts, last_error
		FROM deliveries
		WHERE id = $1
	`, deliveryID).Scan(&status, &attempts, &lastErr); err != nil {
		t.Fatalf("select after MarkRetry() error = %v", err)
	}
	if status != "pending" {
		t.Fatalf("unexpected status after retry: %s", status)
	}
	if attempts != 1 {
		t.Fatalf("unexpected attempts after retry: %d", attempts)
	}
	if !lastErr.Valid || lastErr.String == "" {
		t.Fatal("expected last_error to be set after retry")
	}

	if err := repo.MarkSent(ctx, deliveryID); err != nil {
		t.Fatalf("MarkSent() error = %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT status, last_error
		FROM deliveries
		WHERE id = $1
	`, deliveryID).Scan(&status, &lastErr); err != nil {
		t.Fatalf("select after MarkSent() error = %v", err)
	}
	if status != "sent" {
		t.Fatalf("unexpected status after sent: %s", status)
	}
	if lastErr.Valid {
		t.Fatalf("expected last_error to be NULL after sent, got %q", lastErr.String)
	}

	if err := repo.MarkFailed(ctx, deliveryID, "permanent error"); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT status, attempts, last_error
		FROM deliveries
		WHERE id = $1
	`, deliveryID).Scan(&status, &attempts, &lastErr); err != nil {
		t.Fatalf("select after MarkFailed() error = %v", err)
	}
	if status != "failed" {
		t.Fatalf("unexpected status after failed: %s", status)
	}
	if attempts != 2 {
		t.Fatalf("unexpected attempts after failed: %d", attempts)
	}
}
