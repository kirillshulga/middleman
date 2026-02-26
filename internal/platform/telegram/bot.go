package telegram

import (
	"context"
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
		log.Println("update.Message.Chat.ID: ", update.Message.Chat.ID)

		externalID := strconv.Itoa(update.Message.MessageID)
		createdAt := time.Unix(int64(update.Message.Date), 0)
		_, err = b.msgService.CreateMessageWithDeliveries(
			r.Context(),
			domain.PlatformTelegram,
			externalID,
			sender,
			update.Message.Text,
			createdAt,
		)

		if err != nil && err != service.ErrDuplicateMessage {
			log.Println("telegram error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (b *Bot) Send(ctx context.Context, msg *domain.Message) error {
	//if msg.ChatID == "" {
	//	return fmt.Errorf("chat id is empty")
	//}

	chatID := int64(-5163496194) // TODO: Прокинуть ChatID

	text := msg.Text
	message := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(message)
	return err
}
