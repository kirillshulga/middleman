package domain

import (
	"time"

	"github.com/google/uuid"
)

type DeliveryStatus string

const (
	DeliveryPending    DeliveryStatus = "pending"
	DeliveryProcessing DeliveryStatus = "processing"
	DeliverySent       DeliveryStatus = "sent"
	DeliveryFailed     DeliveryStatus = "failed"
)

type Delivery struct {
	ID               uuid.UUID
	MessageID        uuid.UUID
	TargetEndpointID uuid.UUID
	Status           DeliveryStatus
	Attempts         int
	NextRetryAt      time.Time
	LastError        *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
