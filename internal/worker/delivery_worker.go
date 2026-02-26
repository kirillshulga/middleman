package worker

import (
	"context"
	"log"
	"time"

	"middleman/internal/service"
)

type DeliveryWorker struct {
	service   *service.DeliveryService
	interval  time.Duration
	batchSize int
}

func NewDeliveryWorker(
	service *service.DeliveryService,
	interval time.Duration,
	batchSize int,
) *DeliveryWorker {
	return &DeliveryWorker{
		service:   service,
		interval:  interval,
		batchSize: batchSize,
	}
}

func (w *DeliveryWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Println("Delivery worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Delivery worker stopped")
			return

		case <-ticker.C:
			err := w.service.ProcessPending(ctx, w.batchSize)
			if err != nil {
				log.Printf("worker error: %v\n", err)
			}
		}
	}
}
