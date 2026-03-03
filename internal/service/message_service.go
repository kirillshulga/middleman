package service

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/google/uuid"

	"middleman/internal/domain"
	"middleman/internal/repository"
)

var ErrDuplicateMessage = errors.New("message already exists")
var ErrSourceEndpointNotFound = errors.New("source endpoint not found")

type MessageService struct {
	msgRepo      repository.MessageRepository
	delRepo      repository.DeliveryRepository
	endpointRepo repository.EndpointRepository
	txMgr        TxManager
}

type TxManager interface {
	Begin(ctx context.Context) (repository.Tx, error)
}

func NewMessageService(
	msgRepo repository.MessageRepository,
	delRepo repository.DeliveryRepository,
	endpointRepo repository.EndpointRepository,
	txMgr TxManager,
) *MessageService {
	return &MessageService{
		msgRepo:      msgRepo,
		delRepo:      delRepo,
		endpointRepo: endpointRepo,
		txMgr:        txMgr,
	}
}

func (s *MessageService) CreateMessageWithDeliveries(
	ctx context.Context,
	sourcePlatform domain.Platform,
	sourceChatID string,
	sourceExternalMessageID string,
	sender string,
	text string,
	createdAt time.Time,
) (*domain.Message, error) {
	sourceEndpoint, err := s.endpointRepo.GetByPlatformChatID(ctx, sourcePlatform, sourceChatID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSourceEndpointNotFound
		}
		return nil, err
	}

	tx, err := s.txMgr.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	now := time.Now()

	msg := &domain.Message{
		ID:                      uuid.New(),
		RoomID:                  sourceEndpoint.RoomID,
		SourceEndpointID:        sourceEndpoint.ID,
		SourcePlatform:          sourceEndpoint.Platform,
		SourceExternalMessageID: sourceExternalMessageID,
		Sender:                  sender,
		Text:                    text,
		CreatedAt:               createdAt,
		ReceivedAt:              now,
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
	targetEndpoints, err := s.endpointRepo.ListActiveByRoom(ctx, sourceEndpoint.RoomID)
	if err != nil {
		return nil, err
	}

	deliveries := make([]domain.Delivery, 0, len(targetEndpoints))

	for _, endpoint := range targetEndpoints {
		if endpoint.ID == sourceEndpoint.ID {
			continue
		}

		deliveries = append(deliveries, domain.Delivery{
			ID:               uuid.New(),
			MessageID:        msg.ID,
			TargetEndpointID: endpoint.ID,
			Status:           domain.DeliveryPending,
			Attempts:         0,
			NextRetryAt:      now,
			CreatedAt:        now,
			UpdatedAt:        now,
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
