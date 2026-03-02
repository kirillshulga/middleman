package telegram

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"strconv"
	"time"

	"middleman/internal/domain"
	"middleman/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api        *tgbotapi.BotAPI
	msgService *service.MessageService
}

func NewBot(token string, msgService *service.MessageService) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Bot{
		api:        api,
		msgService: msgService,
	}, nil
}

func (b *Bot) WebhookHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		update, err := b.api.HandleUpdate(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if update.Message == nil || update.Message.Text == "" {
			w.WriteHeader(http.StatusOK)
			return
		}

		sender := update.Message.From.UserName
		if sender == "" {
			sender = update.Message.From.FirstName
		}

		externalID := strconv.Itoa(update.Message.MessageID)
		chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
		createdAt := time.Unix(int64(update.Message.Date), 0)
		_, err = b.msgService.CreateMessageWithDeliveries(
			r.Context(),
			domain.PlatformTelegram,
			chatID,
			externalID,
			sender,
			update.Message.Text,
			createdAt,
		)

		if err != nil && errors.Is(err, service.ErrSourceEndpointNotFound) {
			w.WriteHeader(http.StatusOK)
			return
		}

		if err != nil && err != service.ErrDuplicateMessage {
			log.Println("telegram error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (b *Bot) Send(ctx context.Context, endpoint *domain.Endpoint, msg *domain.Message) error {
	chatID, err := strconv.ParseInt(endpoint.ExternalChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat id: %w", err)
	}

	escapedText := escapeHTML(msg.Text)
	title := fmt.Sprintf("<b>%s in %s:</b> \n", msg.Sender, string(msg.SourcePlatform))
	text := title + escapedText
	message := tgbotapi.NewMessage(chatID, text)
	message.ParseMode = tgbotapi.ModeHTML
	_, err = b.api.Send(message)
	return err
}

func escapeHTML(text string) string {
	return fmt.Sprintf("<blockquote>%s</blockquote>", html.EscapeString(text))
}
