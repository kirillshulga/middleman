package domain

import (
	"time"

	"github.com/google/uuid"
)

type EndpointStatus string

// TODO: Добавить функционал включения / отключения комнат
const (
	EndpointActive   EndpointStatus = "active"
	EndpointDisabled EndpointStatus = "disabled"
)

type Room struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Endpoint struct {
	ID             uuid.UUID
	RoomID         uuid.UUID
	Platform       Platform
	ExternalChatID string
	Status         EndpointStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
