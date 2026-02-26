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
	ID               uuid.UUID
	GlobalSeq        int64
	SourcePlatform   Platform
	SourceExternalID string

	Sender string
	Text   string

	CreatedAt  time.Time
	ReceivedAt time.Time
}
