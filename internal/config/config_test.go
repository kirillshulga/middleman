package config

import (
	"strings"
	"testing"
)

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_KEY", "value")
	if got := getEnv("TEST_KEY", "fallback"); got != "value" {
		t.Fatalf("unexpected value: got %q want %q", got, "value")
	}
	if got := getEnv("MISSING_KEY", "fallback"); got != "fallback" {
		t.Fatalf("unexpected fallback: got %q want %q", got, "fallback")
	}
}

func TestLoad_UsesEnvironmentForDatabaseURLAndPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("TELEGRAM_TOKEN", "tg-token")
	t.Setenv("TELEGRAM_WEBHOOK_SECRET", "tg-secret")
	t.Setenv("TELEGRAM_WEBHOOK_URL", "https://example.com/telegram-webhook")
	t.Setenv("DISCORD_TOKEN", "dc-token")
	t.Setenv("SLACK_TOKEN", "sl-token")
	t.Setenv("SLACK_SIGNING_SECRET", "slack-secret")

	cfg := Load()
	if cfg.DatabaseURL != "postgres://test:test@localhost:5432/test" {
		t.Fatalf("unexpected DatabaseURL: got %q", cfg.DatabaseURL)
	}
	if cfg.HTTPPort != "9090" {
		t.Fatalf("unexpected HTTPPort: got %q", cfg.HTTPPort)
	}
	if cfg.TelegramToken != "tg-token" || cfg.DiscordToken != "dc-token" || cfg.SlackToken != "sl-token" {
		t.Fatalf("unexpected tokens: %#v", cfg)
	}
	if cfg.TelegramWebhookSecret != "tg-secret" || cfg.SlackSigningSecret != "slack-secret" {
		t.Fatalf("unexpected webhook secrets: %#v", cfg)
	}
	if cfg.TelegramWebhookURL != "https://example.com/telegram-webhook" {
		t.Fatalf("unexpected telegram webhook url: %#v", cfg)
	}
}

func TestValidate(t *testing.T) {
	t.Setenv("TELEGRAM_TOKEN", "tg-token")
	t.Setenv("TELEGRAM_WEBHOOK_SECRET", "tg-secret")
	t.Setenv("DISCORD_TOKEN", "dc-token")
	t.Setenv("SLACK_TOKEN", "sl-token")
	t.Setenv("SLACK_SIGNING_SECRET", "slack-secret")
	cfg := Load()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidate_Missing(t *testing.T) {
	cfg := &Config{}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "TELEGRAM_TOKEN") {
		t.Fatalf("unexpected error: %v", err)
	}
}
