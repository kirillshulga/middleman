package service

import (
	"context"
	"middleman/internal/domain"
	"middleman/internal/repository"
)

type DeliveryService struct {
	delRepo repository.DeliveryRepository
	msgRepo repository.MessageRepository
	clients map[domain.Platform]PlatformClient
}

type PlatformClient interface {
	Send(ctx context.Context, msg *domain.Message) error
}

func NewDeliveryService(
	delRepo repository.DeliveryRepository,
	msgRepo repository.MessageRepository,
	clients map[domain.Platform]PlatformClient,
) *DeliveryService {
	return &DeliveryService{
		delRepo: delRepo,
		msgRepo: msgRepo,
		clients: clients,
	}
}

func (s *DeliveryService) ProcessPending(ctx context.Context, limit int) error {
	deliveries, err := s.delRepo.PickPending(ctx, limit)
	if err != nil {
		return err
	}

	for _, d := range deliveries {

		if d.Attempts >= 5 {
			// TODO: Разобраться что делать с попытками
			err = s.delRepo.UpdateStatus(ctx, d.ID, domain.DeliveryFailed, d.LastError)
			continue
		}

		msg, err := s.msgRepo.GetByID(ctx, d.MessageID)

		if err != nil {
			continue
		}

		client := s.clients[d.Platform]
		if client == nil {
			errStr := "platform client not found"
			err = s.delRepo.UpdateStatus(ctx, d.ID, domain.DeliveryFailed, &errStr)
			continue
		}

		err = client.Send(ctx, msg)
		if err != nil {
			errStr := err.Error()
			_ = s.delRepo.UpdateStatus(ctx, d.ID, domain.DeliveryPending, &errStr)
			continue
		}

		_ = s.delRepo.UpdateStatus(ctx, d.ID, domain.DeliverySent, nil)
	}

	return nil
}
