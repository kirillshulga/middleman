package telegram

import "testing"

func TestIsValidTelegramSecret(t *testing.T) {
	t.Parallel()

	if !isValidTelegramSecret("secret", "secret") {
		t.Fatal("expected secret to be valid")
	}
	if isValidTelegramSecret("wrong", "secret") {
		t.Fatal("expected wrong secret to be invalid")
	}
	if isValidTelegramSecret("", "secret") {
		t.Fatal("expected empty secret to be invalid")
	}
	if isValidTelegramSecret("secret", "") {
		t.Fatal("expected empty expected secret to be invalid")
	}
}

func TestSetWebhook_ValidatesSecret(t *testing.T) {
	t.Parallel()

	b := &Bot{webhookSecret: ""}
	err := b.SetWebhook("https://example.com/telegram-webhook", false)
	if err == nil {
		t.Fatal("expected error for empty webhook secret")
	}
}

func TestSetWebhook_ValidatesURL(t *testing.T) {
	t.Parallel()

	b := &Bot{webhookSecret: "secret"}
	err := b.SetWebhook("://bad-url", false)
	if err == nil {
		t.Fatal("expected error for invalid webhook url")
	}
}
