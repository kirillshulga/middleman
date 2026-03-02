package main

import (
	"context"
	"log"
	"middleman/internal/platform/slack"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"middleman/internal/config"
	"middleman/internal/db"
	"middleman/internal/domain"
	"middleman/internal/platform/discord"
	"middleman/internal/platform/telegram"
	"middleman/internal/repository/postgres"
	"middleman/internal/service"
	"middleman/internal/worker"
)

func main() {

	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ===== DB =====
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	// ===== Repositories =====
	msgRepo := postgres.NewMessageRepository(database.Pool)
	delRepo := postgres.NewDeliveryRepository(database.Pool)
	endpointRepo := postgres.NewEndpointRepository(database.Pool)
	txMgr := postgres.NewTxManager(database.Pool)

	// ===== Message Service =====
	msgService := service.NewMessageService(
		msgRepo,
		delRepo,
		endpointRepo,
		txMgr,
	)

	// ===== Telegram Bot =====
	telegramBot, err := telegram.NewBot(cfg.TelegramToken, msgService)
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/telegram-webhook", telegramBot.WebhookHandler())

	// ===== Discord Bot =====
	discordBot, err := discord.NewBot(cfg.DiscordToken, msgService)
	if err != nil {
		log.Fatal(err)
	}

	if err := discordBot.Start(); err != nil {
		log.Fatal(err)
	}
	defer discordBot.Stop()

	// ===== Slack Bot =====
	slackBot := slack.NewBot(cfg.SlackToken, msgService)
	http.HandleFunc("/slack/webhook", slackBot.WebhookHandler())

	clients := make(map[domain.Platform]service.PlatformClient)

	clients[domain.PlatformTelegram] = telegramBot
	clients[domain.PlatformDiscord] = discordBot
	clients[domain.PlatformSlack] = slackBot

	// ===== Delivery Service =====
	delService := service.NewDeliveryService(
		delRepo,
		msgRepo,
		endpointRepo,
		clients,
	)

	// ===== Worker =====
	w := worker.NewDeliveryWorker(delService, 2*time.Second, 10)
	go w.Start(ctx)

	// ===== HTTP Server =====
	server := &http.Server{
		Addr: ":" + cfg.HTTPPort,
	}

	go func() {
		log.Println("HTTP server started on port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// ===== Graceful shutdown =====
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down...")

	cancel()

	shutdownCtx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	server.Shutdown(shutdownCtx)
}
