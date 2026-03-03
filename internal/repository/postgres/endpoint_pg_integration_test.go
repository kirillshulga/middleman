package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"middleman/internal/domain"
)

func TestEndpointRepository_Integration_GetByPlatformChatIDAndListActiveByRoom(t *testing.T) {
	pool := setupIntegrationDB(t)
	repo := NewEndpointRepository(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	roomID := uuid.New()
	activeEndpointID := uuid.New()
	disabledEndpointID := uuid.New()

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
		  ($2, $3, 'slack', 'slack-1', 'disabled', NOW(), NOW())
	`, activeEndpointID, disabledEndpointID, roomID)
	if err != nil {
		t.Fatalf("insert endpoints error = %v", err)
	}

	got, err := repo.GetByPlatformChatID(ctx, domain.PlatformTelegram, "tg-1")
	if err != nil {
		t.Fatalf("GetByPlatformChatID() error = %v", err)
	}
	if got.ID != activeEndpointID {
		t.Fatalf("unexpected endpoint id: got %s want %s", got.ID, activeEndpointID)
	}

	activeEndpoints, err := repo.ListActiveByRoom(ctx, roomID)
	if err != nil {
		t.Fatalf("ListActiveByRoom() error = %v", err)
	}
	if len(activeEndpoints) != 1 {
		t.Fatalf("expected one active endpoint, got %d", len(activeEndpoints))
	}
	if activeEndpoints[0].ID != activeEndpointID {
		t.Fatalf("unexpected active endpoint: got %s want %s", activeEndpoints[0].ID, activeEndpointID)
	}
}
