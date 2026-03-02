package domain

import (
	"time"

	"github.com/google/uuid"
)

type Platform string

const (
	PlatformTelegram Platform = "telegram"
	PlatformSlack    Platform = "slack"
	PlatformDiscord  Platform = "discord"
)

type Message struct {
	ID                      uuid.UUID
	GlobalSeq               int64
	RoomID                  uuid.UUID
	SourceEndpointID        uuid.UUID
	SourcePlatform          Platform
	SourceExternalMessageID string

	Sender string
	Text   string

	CreatedAt  time.Time
	ReceivedAt time.Time
}
