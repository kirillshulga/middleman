package discord

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"

	"middleman/internal/domain"
	"middleman/internal/service"
)

type Bot struct {
	session    *discordgo.Session
	msgService *service.MessageService
}

func NewBot(token string, msgService *service.MessageService) (*Bot, error) {

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		session:    session,
		msgService: msgService,
	}

	session.AddHandler(bot.messageCreate)

	return bot, nil
}

func (b *Bot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.Bot {
		return
	}

	if m.Content == "" {
		return
	}

	log.Println("m.ChannelID:", m.ChannelID)

	ctx := context.Background()

	_, err := b.msgService.CreateMessageWithDeliveries(
		ctx,
		domain.PlatformDiscord,
		m.ID,
		m.Author.Username,
		m.Content,
		m.Timestamp,
	)

	if err != nil && err != service.ErrDuplicateMessage {
		log.Println("discord error:", err)
	}
}

func (b *Bot) Start() error {
	return b.session.Open()
}

func (b *Bot) Stop() error {
	return b.session.Close()
}

func (b *Bot) Send(ctx context.Context, msg *domain.Message) error {
	//if msg.ChatID == "" {
	//	return fmt.Errorf("channel id is empty")
	//}
	ChannelID := "1475705150713630906"
	_, err := b.session.ChannelMessageSend(
		ChannelID,
		msg.Text,
	)

	return err
}
