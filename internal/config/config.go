package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL           string
	TelegramToken         string
	TelegramWebhookSecret string
	TelegramWebhookURL    string
	DiscordToken          string
	HTTPPort              string
	SlackToken            string
	SlackSigningSecret    string

	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration

	AlertInterval            time.Duration
	AlertRetryWindow         time.Duration
	AlertFailedThreshold     int64
	AlertBacklogThreshold    int64
	AlertRetrySpikeThreshold int64
}

func Load() *Config {
	return &Config{
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://sync_user:sync_pass@localhost:5432/sync_db?sslmode=disable"),
		TelegramToken:         getEnv("TELEGRAM_TOKEN", ""),
		TelegramWebhookSecret: getEnv("TELEGRAM_WEBHOOK_SECRET", ""),
		TelegramWebhookURL:    getEnv("TELEGRAM_WEBHOOK_URL", ""),
		DiscordToken:          getEnv("DISCORD_TOKEN", ""),
		SlackToken:            getEnv("SLACK_TOKEN", ""),
		SlackSigningSecret:    getEnv("SLACK_SIGNING_SECRET", ""),
		HTTPPort:              getEnv("HTTP_PORT", "8080"),

		HTTPReadTimeout:  time.Duration(getEnvInt("HTTP_READ_TIMEOUT_SEC", 10)) * time.Second,
		HTTPWriteTimeout: time.Duration(getEnvInt("HTTP_WRITE_TIMEOUT_SEC", 20)) * time.Second,
		HTTPIdleTimeout:  time.Duration(getEnvInt("HTTP_IDLE_TIMEOUT_SEC", 60)) * time.Second,

		AlertInterval:            time.Duration(getEnvInt("DELIVERY_ALERT_INTERVAL_SEC", 30)) * time.Second,
		AlertRetryWindow:         time.Duration(getEnvInt("DELIVERY_ALERT_RETRY_WINDOW_SEC", 300)) * time.Second,
		AlertFailedThreshold:     int64(getEnvInt("DELIVERY_ALERT_FAILED_THRESHOLD", 10)),
		AlertBacklogThreshold:    int64(getEnvInt("DELIVERY_ALERT_BACKLOG_THRESHOLD", 100)),
		AlertRetrySpikeThreshold: int64(getEnvInt("DELIVERY_ALERT_RETRY_SPIKE_THRESHOLD", 30)),
	}
}

func (c *Config) Validate() error {
	missing := make([]string, 0, 5)

	if c.TelegramToken == "" {
		missing = append(missing, "TELEGRAM_TOKEN")
	}
	if c.TelegramWebhookSecret == "" {
		missing = append(missing, "TELEGRAM_WEBHOOK_SECRET")
	}
	if c.DiscordToken == "" {
		missing = append(missing, "DISCORD_TOKEN")
	}
	if c.SlackToken == "" {
		missing = append(missing, "SLACK_TOKEN")
	}
	if c.SlackSigningSecret == "" {
		missing = append(missing, "SLACK_SIGNING_SECRET")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return v
}
