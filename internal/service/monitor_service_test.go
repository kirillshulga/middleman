package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"middleman/internal/domain"
	"middleman/internal/repository"
)

type monitorRepoMock struct {
	stats repository.DeliveryQueueStats
}

func (m *monitorRepoMock) CreateBatch(_ context.Context, _ repository.Tx, _ []domain.Delivery) error {
	return nil
}

func (m *monitorRepoMock) ClaimPending(_ context.Context, _ int) ([]domain.Delivery, error) {
	return nil, nil
}

func (m *monitorRepoMock) MarkSent(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *monitorRepoMock) MarkRetry(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}

func (m *monitorRepoMock) MarkFailed(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (m *monitorRepoMock) GetQueueStats(_ context.Context, _ time.Time) (repository.DeliveryQueueStats, error) {
	return m.stats, nil
}

func TestMonitorService_Evaluate(t *testing.T) {
	t.Parallel()

	svc := NewMonitorService(
		&monitorRepoMock{},
		AlertThresholds{
			FailedThreshold:     10,
			BacklogThreshold:    100,
			RetrySpikeThreshold: 30,
			RetryWindow:         5 * time.Minute,
		},
	)

	alerts := svc.Evaluate(QueueSnapshot{
		FailedCount:         12,
		BacklogPendingCount: 101,
		RetrySpikeCount:     29,
		RetryWindowSec:      300,
	})
	if len(alerts) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(alerts))
	}
}

func TestMonitorService_Snapshot(t *testing.T) {
	t.Parallel()

	repo := &monitorRepoMock{
		stats: repository.DeliveryQueueStats{
			FailedCount:         5,
			BacklogPendingCount: 7,
			RetrySpikeCount:     2,
		},
	}
	svc := NewMonitorService(
		repo,
		AlertThresholds{
			FailedThreshold:     10,
			BacklogThreshold:    100,
			RetrySpikeThreshold: 30,
			RetryWindow:         5 * time.Minute,
		},
	)

	snapshot, err := svc.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if snapshot.FailedCount != 5 || snapshot.BacklogPendingCount != 7 || snapshot.RetrySpikeCount != 2 {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
	if snapshot.RetryWindowSec != 300 {
		t.Fatalf("unexpected retry window: %d", snapshot.RetryWindowSec)
	}
}
