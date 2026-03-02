package service

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"middleman/internal/domain"
	"middleman/internal/repository"
)

type DeliveryService struct {
	delRepo      repository.DeliveryRepository
	msgRepo      repository.MessageRepository
	endpointRepo repository.EndpointRepository
	clients      map[domain.Platform]PlatformClient
	maxAttempts  int
}

type PlatformClient interface {
	Send(ctx context.Context, endpoint *domain.Endpoint, msg *domain.Message) error
}

func NewDeliveryService(
	delRepo repository.DeliveryRepository,
	msgRepo repository.MessageRepository,
	endpointRepo repository.EndpointRepository,
	clients map[domain.Platform]PlatformClient,
) *DeliveryService {
	return &DeliveryService{
		delRepo:      delRepo,
		msgRepo:      msgRepo,
		endpointRepo: endpointRepo,
		clients:      clients,
		maxAttempts:  5,
	}
}

func (s *DeliveryService) ProcessPending(ctx context.Context, limit int) error {
	deliveries, err := s.delRepo.ClaimPending(ctx, limit)
	if err != nil {
		return err
	}

	for _, d := range deliveries {
		if d.Attempts >= s.maxAttempts {
			if err := s.delRepo.MarkFailed(ctx, d.ID, "max attempts reached"); err != nil {
				return err
			}
			continue
		}

		msg, err := s.msgRepo.GetByID(ctx, d.MessageID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				if markErr := s.delRepo.MarkFailed(ctx, d.ID, "message not found"); markErr != nil {
					return markErr
				}
				continue
			}
			return err
		}

		targetEndpoint, err := s.endpointRepo.GetByID(ctx, d.TargetEndpointID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				if markErr := s.delRepo.MarkFailed(ctx, d.ID, "target endpoint not found"); markErr != nil {
					return markErr
				}
				continue
			}
			retryAt := time.Now().Add(backoff(d.Attempts + 1))
			if markErr := s.delRepo.MarkRetry(ctx, d.ID, err.Error(), retryAt); markErr != nil {
				return markErr
			}
			continue
		}

		client := s.clients[targetEndpoint.Platform]
		if client == nil {
			if err := s.delRepo.MarkFailed(ctx, d.ID, "platform client not found"); err != nil {
				return err
			}
			continue
		}

		err = client.Send(ctx, targetEndpoint, msg)
		if err != nil {
			retryAt := time.Now().Add(backoff(d.Attempts + 1))
			if d.Attempts+1 >= s.maxAttempts {
				if markErr := s.delRepo.MarkFailed(ctx, d.ID, err.Error()); markErr != nil {
					return markErr
				}
				continue
			}

			if markErr := s.delRepo.MarkRetry(ctx, d.ID, err.Error(), retryAt); markErr != nil {
				return markErr
			}
			continue
		}

		if err := s.delRepo.MarkSent(ctx, d.ID); err != nil {
			return err
		}
	}

	return nil
}

func backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	return time.Duration(attempt*5) * time.Second
}
