package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"middleman/internal/domain"
	"middleman/internal/repository"
)

type msgRepoMock struct {
	createFn func(ctx context.Context, tx repository.Tx, msg *domain.Message) error
}

func (m *msgRepoMock) Create(ctx context.Context, tx repository.Tx, msg *domain.Message) error {
	if m.createFn != nil {
		return m.createFn(ctx, tx, msg)
	}
	return nil
}

func (m *msgRepoMock) GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	return nil, errors.New("not implemented")
}

type delRepoMock struct {
	createBatchFn func(ctx context.Context, tx repository.Tx, deliveries []domain.Delivery) error
}

func (m *delRepoMock) CreateBatch(ctx context.Context, tx repository.Tx, deliveries []domain.Delivery) error {
	if m.createBatchFn != nil {
		return m.createBatchFn(ctx, tx, deliveries)
	}
	return nil
}

func (m *delRepoMock) ClaimPending(ctx context.Context, limit int) ([]domain.Delivery, error) {
	return nil, errors.New("not implemented")
}

func (m *delRepoMock) MarkSent(ctx context.Context, id uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *delRepoMock) MarkRetry(ctx context.Context, id uuid.UUID, lastErr string, nextRetryAt time.Time) error {
	return errors.New("not implemented")
}

func (m *delRepoMock) MarkFailed(ctx context.Context, id uuid.UUID, lastErr string) error {
	return errors.New("not implemented")
}

func (m *delRepoMock) GetQueueStats(ctx context.Context, retrySince time.Time) (repository.DeliveryQueueStats, error) {
	return repository.DeliveryQueueStats{}, errors.New("not implemented")
}

type endpointRepoMock struct {
	getByPlatformChatIDFn func(ctx context.Context, platform domain.Platform, externalChatID string) (*domain.Endpoint, error)
	listActiveByRoomFn    func(ctx context.Context, roomID uuid.UUID) ([]domain.Endpoint, error)
}

func (m *endpointRepoMock) GetByID(ctx context.Context, id uuid.UUID) (*domain.Endpoint, error) {
	return nil, errors.New("not implemented")
}

func (m *endpointRepoMock) GetByPlatformChatID(ctx context.Context, platform domain.Platform, externalChatID string) (*domain.Endpoint, error) {
	if m.getByPlatformChatIDFn != nil {
		return m.getByPlatformChatIDFn(ctx, platform, externalChatID)
	}
	return nil, pgx.ErrNoRows
}

func (m *endpointRepoMock) ListActiveByRoom(ctx context.Context, roomID uuid.UUID) ([]domain.Endpoint, error) {
	if m.listActiveByRoomFn != nil {
		return m.listActiveByRoomFn(ctx, roomID)
	}
	return nil, nil
}

type txMock struct {
	committed bool
	rolled    bool
	commitErr error
}

func (m *txMock) Commit(ctx context.Context) error {
	m.committed = true
	return m.commitErr
}

func (m *txMock) Rollback(ctx context.Context) error {
	m.rolled = true
	return nil
}

type txMgrMock struct {
	tx  repository.Tx
	err error
}

func (m *txMgrMock) Begin(ctx context.Context) (repository.Tx, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tx, nil
}

func TestMessageService_CreateMessageWithDeliveries_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	roomID := uuid.New()
	sourceEndpointID := uuid.New()
	targetEndpointID := uuid.New()
	createdAt := time.Now().Add(-1 * time.Minute)
	tx := &txMock{}

	sourceEndpoint := &domain.Endpoint{
		ID:             sourceEndpointID,
		RoomID:         roomID,
		Platform:       domain.PlatformTelegram,
		ExternalChatID: "123",
		Status:         domain.EndpointActive,
	}

	var gotDeliveries []domain.Delivery
	msgRepo := &msgRepoMock{
		createFn: func(ctx context.Context, tx repository.Tx, msg *domain.Message) error {
			msg.GlobalSeq = 42
			return nil
		},
	}
	delRepo := &delRepoMock{
		createBatchFn: func(ctx context.Context, tx repository.Tx, deliveries []domain.Delivery) error {
			gotDeliveries = deliveries
			return nil
		},
	}
	endpointRepo := &endpointRepoMock{
		getByPlatformChatIDFn: func(ctx context.Context, platform domain.Platform, externalChatID string) (*domain.Endpoint, error) {
			return sourceEndpoint, nil
		},
		listActiveByRoomFn: func(ctx context.Context, roomID uuid.UUID) ([]domain.Endpoint, error) {
			return []domain.Endpoint{
				*sourceEndpoint,
				{
					ID:             targetEndpointID,
					RoomID:         roomID,
					Platform:       domain.PlatformSlack,
					ExternalChatID: "C123",
					Status:         domain.EndpointActive,
				},
			}, nil
		},
	}
	svc := NewMessageService(msgRepo, delRepo, endpointRepo, &txMgrMock{tx: tx})

	msg, err := svc.CreateMessageWithDeliveries(
		ctx,
		domain.PlatformTelegram,
		"123",
		"msg-1",
		"alice",
		"hello",
		createdAt,
	)
	if err != nil {
		t.Fatalf("CreateMessageWithDeliveries() error = %v", err)
	}

	if !tx.committed {
		t.Fatal("expected transaction commit")
	}
	if !tx.rolled {
		t.Fatal("expected deferred rollback call")
	}
	if msg.RoomID != roomID {
		t.Fatalf("unexpected room id: got %s want %s", msg.RoomID, roomID)
	}
	if msg.SourceEndpointID != sourceEndpointID {
		t.Fatalf("unexpected source endpoint id: got %s want %s", msg.SourceEndpointID, sourceEndpointID)
	}
	if msg.GlobalSeq != 42 {
		t.Fatalf("unexpected global seq: got %d want 42", msg.GlobalSeq)
	}
	if len(gotDeliveries) != 1 {
		t.Fatalf("expected one delivery, got %d", len(gotDeliveries))
	}
	if gotDeliveries[0].TargetEndpointID != targetEndpointID {
		t.Fatalf("unexpected target endpoint id: got %s want %s", gotDeliveries[0].TargetEndpointID, targetEndpointID)
	}
}

func TestMessageService_CreateMessageWithDeliveries_SourceEndpointNotFound(t *testing.T) {
	t.Parallel()

	svc := NewMessageService(
		&msgRepoMock{},
		&delRepoMock{},
		&endpointRepoMock{
			getByPlatformChatIDFn: func(ctx context.Context, platform domain.Platform, externalChatID string) (*domain.Endpoint, error) {
				return nil, pgx.ErrNoRows
			},
		},
		&txMgrMock{tx: &txMock{}},
	)

	_, err := svc.CreateMessageWithDeliveries(
		context.Background(),
		domain.PlatformTelegram,
		"unknown-chat",
		"msg-1",
		"alice",
		"hello",
		time.Now(),
	)
	if !errors.Is(err, ErrSourceEndpointNotFound) {
		t.Fatalf("expected ErrSourceEndpointNotFound, got %v", err)
	}
}

func TestMessageService_CreateMessageWithDeliveries_DuplicateMessage(t *testing.T) {
	t.Parallel()

	sourceEndpoint := &domain.Endpoint{
		ID:             uuid.New(),
		RoomID:         uuid.New(),
		Platform:       domain.PlatformTelegram,
		ExternalChatID: "123",
		Status:         domain.EndpointActive,
	}

	svc := NewMessageService(
		&msgRepoMock{
			createFn: func(ctx context.Context, tx repository.Tx, msg *domain.Message) error {
				return &pgconn.PgError{Code: "23505"}
			},
		},
		&delRepoMock{},
		&endpointRepoMock{
			getByPlatformChatIDFn: func(ctx context.Context, platform domain.Platform, externalChatID string) (*domain.Endpoint, error) {
				return sourceEndpoint, nil
			},
		},
		&txMgrMock{tx: &txMock{}},
	)

	_, err := svc.CreateMessageWithDeliveries(
		context.Background(),
		domain.PlatformTelegram,
		"123",
		"msg-1",
		"alice",
		"hello",
		time.Now(),
	)
	if !errors.Is(err, ErrDuplicateMessage) {
		t.Fatalf("expected ErrDuplicateMessage, got %v", err)
	}
}

func TestMessageService_CreateMessageWithDeliveries_CreateBatchError(t *testing.T) {
	t.Parallel()

	roomID := uuid.New()
	sourceEndpoint := &domain.Endpoint{
		ID:             uuid.New(),
		RoomID:         roomID,
		Platform:       domain.PlatformTelegram,
		ExternalChatID: "123",
		Status:         domain.EndpointActive,
	}
	tx := &txMock{}
	wantErr := errors.New("create batch failed")

	svc := NewMessageService(
		&msgRepoMock{
			createFn: func(ctx context.Context, tx repository.Tx, msg *domain.Message) error {
				return nil
			},
		},
		&delRepoMock{
			createBatchFn: func(ctx context.Context, tx repository.Tx, deliveries []domain.Delivery) error {
				return wantErr
			},
		},
		&endpointRepoMock{
			getByPlatformChatIDFn: func(ctx context.Context, platform domain.Platform, externalChatID string) (*domain.Endpoint, error) {
				return sourceEndpoint, nil
			},
			listActiveByRoomFn: func(ctx context.Context, roomID uuid.UUID) ([]domain.Endpoint, error) {
				return []domain.Endpoint{
					*sourceEndpoint,
					{
						ID:             uuid.New(),
						RoomID:         roomID,
						Platform:       domain.PlatformSlack,
						ExternalChatID: "C123",
						Status:         domain.EndpointActive,
					},
				}, nil
			},
		},
		&txMgrMock{tx: tx},
	)

	_, err := svc.CreateMessageWithDeliveries(
		context.Background(),
		domain.PlatformTelegram,
		"123",
		"msg-1",
		"alice",
		"hello",
		time.Now(),
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if tx.committed {
		t.Fatal("transaction should not commit on error")
	}
}
