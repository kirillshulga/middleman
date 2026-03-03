package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"middleman/internal/domain"
	"middleman/internal/repository"
)

type deliveryRepoMock struct {
	claimPendingFn func(ctx context.Context, limit int) ([]domain.Delivery, error)
	markSentFn     func(ctx context.Context, id uuid.UUID) error
	markRetryFn    func(ctx context.Context, id uuid.UUID, lastErr string, nextRetryAt time.Time) error
	markFailedFn   func(ctx context.Context, id uuid.UUID, lastErr string) error
}

func (m *deliveryRepoMock) CreateBatch(_ context.Context, _ repository.Tx, _ []domain.Delivery) error {
	return errors.New("not implemented")
}

func (m *deliveryRepoMock) ClaimPending(ctx context.Context, limit int) ([]domain.Delivery, error) {
	if m.claimPendingFn != nil {
		return m.claimPendingFn(ctx, limit)
	}
	return nil, nil
}

func (m *deliveryRepoMock) MarkSent(ctx context.Context, id uuid.UUID) error {
	if m.markSentFn != nil {
		return m.markSentFn(ctx, id)
	}
	return nil
}

func (m *deliveryRepoMock) MarkRetry(ctx context.Context, id uuid.UUID, lastErr string, nextRetryAt time.Time) error {
	if m.markRetryFn != nil {
		return m.markRetryFn(ctx, id, lastErr, nextRetryAt)
	}
	return nil
}

func (m *deliveryRepoMock) MarkFailed(ctx context.Context, id uuid.UUID, lastErr string) error {
	if m.markFailedFn != nil {
		return m.markFailedFn(ctx, id, lastErr)
	}
	return nil
}

func (m *deliveryRepoMock) GetQueueStats(_ context.Context, _ time.Time) (repository.DeliveryQueueStats, error) {
	return repository.DeliveryQueueStats{}, nil
}

type deliveryMsgRepoMock struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*domain.Message, error)
}

func (m *deliveryMsgRepoMock) Create(_ context.Context, _ repository.Tx, _ *domain.Message) error {
	return errors.New("not implemented")
}

func (m *deliveryMsgRepoMock) GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, pgx.ErrNoRows
}

type deliveryEndpointRepoMock struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*domain.Endpoint, error)
}

func (m *deliveryEndpointRepoMock) GetByID(ctx context.Context, id uuid.UUID) (*domain.Endpoint, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, pgx.ErrNoRows
}

func (m *deliveryEndpointRepoMock) GetByPlatformChatID(_ context.Context, _ domain.Platform, _ string) (*domain.Endpoint, error) {
	return nil, errors.New("not implemented")
}

func (m *deliveryEndpointRepoMock) ListActiveByRoom(_ context.Context, _ uuid.UUID) ([]domain.Endpoint, error) {
	return nil, errors.New("not implemented")
}

type platformClientMock struct {
	sendFn func(ctx context.Context, endpoint *domain.Endpoint, msg *domain.Message) error
}

func (m *platformClientMock) Send(ctx context.Context, endpoint *domain.Endpoint, msg *domain.Message) error {
	if m.sendFn != nil {
		return m.sendFn(ctx, endpoint, msg)
	}
	return nil
}

func TestDeliveryService_ProcessPending_ClaimError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("claim failed")
	svc := NewDeliveryService(
		&deliveryRepoMock{
			claimPendingFn: func(_ context.Context, _ int) ([]domain.Delivery, error) {
				return nil, wantErr
			},
		},
		&deliveryMsgRepoMock{},
		&deliveryEndpointRepoMock{},
		nil,
	)

	err := svc.ProcessPending(context.Background(), 10)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestDeliveryService_ProcessPending_MarkFailedOnMaxAttempts(t *testing.T) {
	t.Parallel()

	deliveryID := uuid.New()
	failedCalled := false
	svc := NewDeliveryService(
		&deliveryRepoMock{
			claimPendingFn: func(_ context.Context, _ int) ([]domain.Delivery, error) {
				return []domain.Delivery{
					{ID: deliveryID, Attempts: 5},
				}, nil
			},
			markFailedFn: func(_ context.Context, id uuid.UUID, _ string) error {
				failedCalled = true
				if id != deliveryID {
					t.Fatalf("unexpected id: %s", id)
				}
				return nil
			},
		},
		&deliveryMsgRepoMock{},
		&deliveryEndpointRepoMock{},
		nil,
	)

	if err := svc.ProcessPending(context.Background(), 10); err != nil {
		t.Fatalf("ProcessPending() error = %v", err)
	}
	if !failedCalled {
		t.Fatal("expected MarkFailed to be called")
	}
}

func TestDeliveryService_ProcessPending_SuccessPath(t *testing.T) {
	t.Parallel()

	deliveryID := uuid.New()
	msgID := uuid.New()
	endpointID := uuid.New()
	sentCalled := false
	clientCalled := false

	svc := NewDeliveryService(
		&deliveryRepoMock{
			claimPendingFn: func(_ context.Context, _ int) ([]domain.Delivery, error) {
				return []domain.Delivery{
					{
						ID:               deliveryID,
						MessageID:        msgID,
						TargetEndpointID: endpointID,
						Attempts:         0,
					},
				}, nil
			},
			markSentFn: func(_ context.Context, id uuid.UUID) error {
				sentCalled = true
				if id != deliveryID {
					t.Fatalf("unexpected id: %s", id)
				}
				return nil
			},
		},
		&deliveryMsgRepoMock{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Message, error) {
				return &domain.Message{
					ID:             msgID,
					SourcePlatform: domain.PlatformTelegram,
					Sender:         "alice",
					Text:           "hello",
				}, nil
			},
		},
		&deliveryEndpointRepoMock{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Endpoint, error) {
				return &domain.Endpoint{
					ID:             endpointID,
					Platform:       domain.PlatformSlack,
					ExternalChatID: "C123",
					Status:         domain.EndpointActive,
				}, nil
			},
		},
		map[domain.Platform]PlatformClient{
			domain.PlatformSlack: &platformClientMock{
				sendFn: func(_ context.Context, _ *domain.Endpoint, _ *domain.Message) error {
					clientCalled = true
					return nil
				},
			},
		},
	)

	if err := svc.ProcessPending(context.Background(), 1); err != nil {
		t.Fatalf("ProcessPending() error = %v", err)
	}
	if !clientCalled {
		t.Fatal("expected client Send call")
	}
	if !sentCalled {
		t.Fatal("expected MarkSent call")
	}
}

func TestDeliveryService_ProcessPending_RetryOnSendError(t *testing.T) {
	t.Parallel()

	deliveryID := uuid.New()
	msgID := uuid.New()
	endpointID := uuid.New()
	retryCalled := false

	svc := NewDeliveryService(
		&deliveryRepoMock{
			claimPendingFn: func(_ context.Context, _ int) ([]domain.Delivery, error) {
				return []domain.Delivery{
					{
						ID:               deliveryID,
						MessageID:        msgID,
						TargetEndpointID: endpointID,
						Attempts:         1,
					},
				}, nil
			},
			markRetryFn: func(_ context.Context, id uuid.UUID, lastErr string, nextRetryAt time.Time) error {
				retryCalled = true
				if id != deliveryID {
					t.Fatalf("unexpected id: %s", id)
				}
				if lastErr == "" {
					t.Fatal("expected non-empty error message")
				}
				if nextRetryAt.Before(time.Now()) {
					t.Fatal("next retry should be in the future")
				}
				return nil
			},
		},
		&deliveryMsgRepoMock{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Message, error) {
				return &domain.Message{ID: msgID}, nil
			},
		},
		&deliveryEndpointRepoMock{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Endpoint, error) {
				return &domain.Endpoint{
					ID:             endpointID,
					Platform:       domain.PlatformSlack,
					ExternalChatID: "C123",
				}, nil
			},
		},
		map[domain.Platform]PlatformClient{
			domain.PlatformSlack: &platformClientMock{
				sendFn: func(_ context.Context, _ *domain.Endpoint, _ *domain.Message) error {
					return errors.New("temporary send failure")
				},
			},
		},
	)

	if err := svc.ProcessPending(context.Background(), 1); err != nil {
		t.Fatalf("ProcessPending() error = %v", err)
	}
	if !retryCalled {
		t.Fatal("expected MarkRetry call")
	}
}

func TestDeliveryService_ProcessPending_FailOnMissingClient(t *testing.T) {
	t.Parallel()

	deliveryID := uuid.New()
	msgID := uuid.New()
	endpointID := uuid.New()
	failedCalled := false

	svc := NewDeliveryService(
		&deliveryRepoMock{
			claimPendingFn: func(_ context.Context, _ int) ([]domain.Delivery, error) {
				return []domain.Delivery{
					{
						ID:               deliveryID,
						MessageID:        msgID,
						TargetEndpointID: endpointID,
					},
				}, nil
			},
			markFailedFn: func(_ context.Context, _ uuid.UUID, _ string) error {
				failedCalled = true
				return nil
			},
		},
		&deliveryMsgRepoMock{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Message, error) {
				return &domain.Message{ID: msgID}, nil
			},
		},
		&deliveryEndpointRepoMock{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Endpoint, error) {
				return &domain.Endpoint{
					ID:       endpointID,
					Platform: domain.PlatformSlack,
				}, nil
			},
		},
		map[domain.Platform]PlatformClient{},
	)

	if err := svc.ProcessPending(context.Background(), 1); err != nil {
		t.Fatalf("ProcessPending() error = %v", err)
	}
	if !failedCalled {
		t.Fatal("expected MarkFailed call")
	}
}

func TestBackoff(t *testing.T) {
	t.Parallel()

	if got := backoff(0); got != 5*time.Second {
		t.Fatalf("unexpected backoff for attempt 0: %v", got)
	}
	if got := backoff(3); got != 15*time.Second {
		t.Fatalf("unexpected backoff for attempt 3: %v", got)
	}
}
