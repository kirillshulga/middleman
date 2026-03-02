package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"middleman/internal/domain"
	"middleman/internal/service"
	"net/http"
	"time"

	"github.com/slack-go/slack"
)

type Bot struct {
	api        *slack.Client
	msgService *service.MessageService
}

type slackEvent struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
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

func NewBot(token string, msgService *service.MessageService) *Bot {
	api := slack.New(token)

	return &Bot{
		api:        api,
		msgService: msgService,
	}
}

// TODO: Slack webhook не проверяет подпись запроса (X-Slack-Signature), endpoint можно подделать.
func (b *Bot) WebhookHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var event slackEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
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

		// создаём сообщение в системе
		_, err := b.msgService.CreateMessageWithDeliveries(
			r.Context(),
			domain.PlatformSlack,
			event.Event.ClientMsgId,
			event.Event.User,
			event.Event.Text,
			createdAt,
		)
		if err != nil && err != service.ErrDuplicateMessage {
			log.Println("slack error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (b *Bot) Send(ctx context.Context, msg *domain.Message) error {
	//if msg.ChatID == "" {
	//	return fmt.Errorf("channel id is empty")
	//}
	chatID := "C0AH4FPN1JR"
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
