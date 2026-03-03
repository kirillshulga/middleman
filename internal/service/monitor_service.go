package service

import (
	"context"
	"fmt"
	"time"

	"middleman/internal/repository"
)

type AlertThresholds struct {
	FailedThreshold     int64
	BacklogThreshold    int64
	RetrySpikeThreshold int64
	RetryWindow         time.Duration
}

type QueueSnapshot struct {
	FailedCount         int64     `json:"failed_count"`
	BacklogPendingCount int64     `json:"backlog_pending_count"`
	RetrySpikeCount     int64     `json:"retry_spike_count"`
	RetryWindowSec      int64     `json:"retry_window_sec"`
	ObservedAt          time.Time `json:"observed_at"`
}

type MonitorService struct {
	delRepo     repository.DeliveryRepository
	thresholds  AlertThresholds
	nowProvider func() time.Time
}

func NewMonitorService(delRepo repository.DeliveryRepository, thresholds AlertThresholds) *MonitorService {
	return &MonitorService{
		delRepo:     delRepo,
		thresholds:  thresholds,
		nowProvider: time.Now,
	}
}

func (s *MonitorService) Snapshot(ctx context.Context) (QueueSnapshot, error) {
	now := s.nowProvider()
	stats, err := s.delRepo.GetQueueStats(ctx, now.Add(-s.thresholds.RetryWindow))
	if err != nil {
		return QueueSnapshot{}, err
	}

	return QueueSnapshot{
		FailedCount:         stats.FailedCount,
		BacklogPendingCount: stats.BacklogPendingCount,
		RetrySpikeCount:     stats.RetrySpikeCount,
		RetryWindowSec:      int64(s.thresholds.RetryWindow.Seconds()),
		ObservedAt:          now.UTC(),
	}, nil
}

func (s *MonitorService) Evaluate(snapshot QueueSnapshot) []string {
	alerts := make([]string, 0, 3)

	if snapshot.FailedCount >= s.thresholds.FailedThreshold {
		alerts = append(alerts, fmt.Sprintf("failed deliveries threshold exceeded: %d >= %d", snapshot.FailedCount, s.thresholds.FailedThreshold))
	}
	if snapshot.BacklogPendingCount >= s.thresholds.BacklogThreshold {
		alerts = append(alerts, fmt.Sprintf("pending backlog threshold exceeded: %d >= %d", snapshot.BacklogPendingCount, s.thresholds.BacklogThreshold))
	}
	if snapshot.RetrySpikeCount >= s.thresholds.RetrySpikeThreshold {
		alerts = append(alerts, fmt.Sprintf("retry spike threshold exceeded: %d >= %d in %ds", snapshot.RetrySpikeCount, s.thresholds.RetrySpikeThreshold, snapshot.RetryWindowSec))
	}

	return alerts
}
