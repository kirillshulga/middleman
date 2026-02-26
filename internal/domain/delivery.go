package domain

import (
	"time"

	"github.com/google/uuid"
)

type DeliveryStatus string

const (
	DeliveryPending DeliveryStatus = "pending"
	DeliverySent    DeliveryStatus = "sent"
	DeliveryFailed  DeliveryStatus = "failed"
)

type Delivery struct {
	ID        uuid.UUID
	MessageID uuid.UUID
	Platform  Platform
	Status    DeliveryStatus
	Attempts  int
	LastError *string
	CreatedAt time.Time
	UpdatedAt time.Time
}
