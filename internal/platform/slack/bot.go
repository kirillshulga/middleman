package slack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"middleman/internal/domain"
	"middleman/internal/service"
	"net/http"
	"strconv"
	"time"

	"github.com/slack-go/slack"
)

type Bot struct {
	api           *slack.Client
	msgService    *service.MessageService
	signingSecret string
}

type slackEvent struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
	EventID   string `json:"event_id"`
	EventTime int64  `json:"event_time"`
	Event     struct {
		Type        string `json:"type"`
		User        string `json:"user"`
		ClientMsgId string `json:"client_msg_id"`
		Text        string `json:"text"`
		Channel     string `json:"channel"`
		BotID       string `json:"bot_id"`
	} `json:"event"`
}

func NewBot(token string, signingSecret string, msgService *service.MessageService) *Bot {
	api := slack.New(token)

	return &Bot{
		api:           api,
		msgService:    msgService,
		signingSecret: signingSecret,
	}
}

func (b *Bot) WebhookHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request", http.StatusBadRequest)
			return
		}

		if !b.verifyRequest(r, body) {
			http.Error(w, "invalid slack signature", http.StatusUnauthorized)
			return
		}

		var event slackEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Slack URL verification
		if event.Type == "url_verification" {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(event.Challenge))
			return
		}

		// интересуют только сообщения
		if event.Event.Type != "message" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// игнорируем сообщения от ботов
		if event.Event.BotID != "" {
			w.WriteHeader(http.StatusOK)
			return
		}

		createdAt := time.Unix(event.EventTime, 0)
		sourceExternalMessageID := event.Event.ClientMsgId
		if sourceExternalMessageID == "" {
			sourceExternalMessageID = event.EventID
		}
		if sourceExternalMessageID == "" {
			sourceExternalMessageID = fmt.Sprintf("%s:%d:%s", event.Event.Channel, event.EventTime, event.Event.Text)
		}

		// создаём сообщение в системе
		_, err = b.msgService.CreateMessageWithDeliveries(
			r.Context(),
			domain.PlatformSlack,
			event.Event.Channel,
			sourceExternalMessageID,
			event.Event.User,
			event.Event.Text,
			createdAt,
		)
		if err != nil && errors.Is(err, service.ErrSourceEndpointNotFound) {
			w.WriteHeader(http.StatusOK)
			return
		}

		if err != nil && err != service.ErrDuplicateMessage {
			log.Println("slack error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (b *Bot) verifyRequest(r *http.Request, body []byte) bool {
	if b.signingSecret == "" {
		return false
	}

	signature := r.Header.Get("X-Slack-Signature")
	timestampStr := r.Header.Get("X-Slack-Request-Timestamp")
	if signature == "" || timestampStr == "" {
		return false
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return false
	}

	now := time.Now().Unix()
	if abs(now-timestamp) > 5*60 {
		return false
	}

	baseString := "v0:" + timestampStr + ":" + string(body)
	mac := hmac.New(sha256.New, []byte(b.signingSecret))
	_, _ = mac.Write([]byte(baseString))
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func (b *Bot) Send(ctx context.Context, endpoint *domain.Endpoint, msg *domain.Message) error {
	chatID := endpoint.ExternalChatID
	if msg.Text == "" {
		return fmt.Errorf("message content is empty")
	}

	text := "*" + msg.Sender + "* *in* *" + string(msg.SourcePlatform) + ":* \n" + ">" + msg.Text

	_, _, err := b.api.PostMessageContext(
		ctx,
		chatID,
		slack.MsgOptionText(text, false),
	)

	return err
}
