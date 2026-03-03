package main

import (
	"context"
	"encoding/json"
	"errors"
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
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ===== DB =====
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Pool.Close()

	// ===== Repositories =====
	msgRepo := postgres.NewMessageRepository(database.Pool)
	delRepo := postgres.NewDeliveryRepository(database.Pool)
	endpointRepo := postgres.NewEndpointRepository(database.Pool)
	txMgr := postgres.NewTxManager(database.Pool)
	monitorService := service.NewMonitorService(delRepo, service.AlertThresholds{
		FailedThreshold:     cfg.AlertFailedThreshold,
		BacklogThreshold:    cfg.AlertBacklogThreshold,
		RetrySpikeThreshold: cfg.AlertRetrySpikeThreshold,
		RetryWindow:         cfg.AlertRetryWindow,
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
		})
	})

	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		readyCtx, readyCancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer readyCancel()

		if err := database.Pool.Ping(readyCtx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "not_ready",
				"error":  err.Error(),
			})
			return
		}

		snapshot, err := monitorService.Snapshot(readyCtx)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "not_ready",
				"error":  err.Error(),
			})
			return
		}

		alerts := monitorService.Evaluate(snapshot)
		state := "ready"
		if len(alerts) > 0 {
			state = "degraded"
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": state,
			"queue":  snapshot,
			"alerts": alerts,
		})
	})

	// ===== Message Service =====
	msgService := service.NewMessageService(
		msgRepo,
		delRepo,
		endpointRepo,
		txMgr,
	)

	// ===== Telegram Bot =====
	telegramBot, err := telegram.NewBot(cfg.TelegramToken, cfg.TelegramWebhookSecret, msgService)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.TelegramWebhookURL != "" {
		if err := telegramBot.SetWebhook(cfg.TelegramWebhookURL, false); err != nil {
			log.Fatal(err)
		}
		log.Println("Telegram webhook configured")
	} else {
		log.Println("TELEGRAM_WEBHOOK_URL is empty, skipping Telegram webhook setup")
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
	defer func() {
		if err := discordBot.Stop(); err != nil {
			log.Printf("discord stop error: %v\n", err)
		}
	}()

	// ===== Slack Bot =====
	slackBot := slack.NewBot(cfg.SlackToken, cfg.SlackSigningSecret, msgService)
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
	alertWorker := worker.NewAlertWorker(monitorService, cfg.AlertInterval)
	go alertWorker.Start(ctx)

	// ===== HTTP Server =====
	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		ReadHeaderTimeout: cfg.HTTPReadTimeout,
		ReadTimeout:       cfg.HTTPReadTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	go func() {
		log.Println("HTTP server started on port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	// ===== Graceful shutdown =====
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v\n", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("json encode error: %v\n", err)
	}
}
