package worker

import (
	"context"
	"log"
	"strings"
	"time"

	"middleman/internal/service"
)

type QueueMonitor interface {
	Snapshot(ctx context.Context) (service.QueueSnapshot, error)
	Evaluate(snapshot service.QueueSnapshot) []string
}

type AlertWorker struct {
	monitor  QueueMonitor
	interval time.Duration
}

func NewAlertWorker(monitor QueueMonitor, interval time.Duration) *AlertWorker {
	return &AlertWorker{
		monitor:  monitor,
		interval: interval,
	}
}

func (w *AlertWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Println("Alert worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Alert worker stopped")
			return
		case <-ticker.C:
			snapshot, err := w.monitor.Snapshot(ctx)
			if err != nil {
				log.Printf("alert monitor snapshot error: %v\n", err)
				continue
			}

			alerts := w.monitor.Evaluate(snapshot)
			if len(alerts) == 0 {
				continue
			}

			log.Printf(
				"ALERT: %s | failed=%d backlog=%d retry_spike=%d window_sec=%d\n",
				strings.Join(alerts, "; "),
				snapshot.FailedCount,
				snapshot.BacklogPendingCount,
				snapshot.RetrySpikeCount,
				snapshot.RetryWindowSec,
			)
		}
	}
}
