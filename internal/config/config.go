package config

import (
	"os"
)

type Config struct {
	DatabaseURL   string
	TelegramToken string
	DiscordToken  string
	HTTPPort      string
	SlackToken    string
}

func Load() *Config {
	return &Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		TelegramToken: os.Getenv("TELEGRAM_TOKEN"),
		DiscordToken:  os.Getenv("DISCORD_TOKEN"),
		SlackToken:    os.Getenv("SLACK_TOKEN"),
		HTTPPort:      getEnv("HTTP_PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
