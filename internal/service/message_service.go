package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"middleman/internal/domain"
	"middleman/internal/repository"
)

var ErrDuplicateMessage = errors.New("message already exists")

type MessageService struct {
	msgRepo         repository.MessageRepository
	delRepo         repository.DeliveryRepository
	txMgr           TxManager
	targetPlatforms []domain.Platform
}

type TxManager interface {
	Begin(ctx context.Context) (repository.Tx, error)
}

func NewMessageService(
	msgRepo repository.MessageRepository,
	delRepo repository.DeliveryRepository,
	txMgr TxManager,
	targetPlatforms []domain.Platform,
) *MessageService {
	return &MessageService{
		msgRepo:         msgRepo,
		delRepo:         delRepo,
		txMgr:           txMgr,
		targetPlatforms: targetPlatforms,
	}
}

func (s *MessageService) CreateMessageWithDeliveries(
	ctx context.Context,
	sourcePlatform domain.Platform,
	externalID string,
	sender string,
	text string,
	createdAt time.Time,
) (*domain.Message, error) {

	tx, err := s.txMgr.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	msg := &domain.Message{
		ID:               uuid.New(),
		SourcePlatform:   sourcePlatform,
		SourceExternalID: externalID,
		Sender:           sender,
		Text:             text,
		CreatedAt:        createdAt,
		ReceivedAt:       now,
	}

	// INSERT (global_seq получаем через RETURNING)
	err = s.msgRepo.Create(ctx, tx, msg)
	if err != nil {

		// Проверка на duplicate key
		if isUniqueViolation(err) {
			return nil, ErrDuplicateMessage
		}

		return nil, err
	}

	// Создаём delivery задачи
	deliveries := make([]domain.Delivery, 0, len(s.targetPlatforms))

	for _, platform := range s.targetPlatforms {
		if platform == sourcePlatform {
			continue
		}

		deliveries = append(deliveries, domain.Delivery{
			ID:        uuid.New(),
			MessageID: msg.ID,
			Platform:  platform,
			Status:    domain.DeliveryPending,
			Attempts:  0,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	err = s.delRepo.CreateBatch(ctx, tx, deliveries)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return msg, nil
}
