package slack

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestVerifyRequest(t *testing.T) {
	t.Parallel()

	signingSecret := "test-secret"
	body := []byte(`{"type":"event_callback"}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	base := "v0:" + ts + ":" + string(body)
	mac := hmac.New(sha256.New, []byte(signingSecret))
	_, _ = mac.Write([]byte(base))
	signature := "v0=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest("POST", "/slack/webhook", nil)
	req.Header.Set("X-Slack-Signature", signature)
	req.Header.Set("X-Slack-Request-Timestamp", ts)

	bot := &Bot{signingSecret: signingSecret}
	if !bot.verifyRequest(req, body) {
		t.Fatal("expected request to be valid")
	}
}

func TestVerifyRequest_InvalidSignature(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("POST", "/slack/webhook", nil)
	req.Header.Set("X-Slack-Signature", "v0=invalid")
	req.Header.Set("X-Slack-Request-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))

	bot := &Bot{signingSecret: "secret"}
	if bot.verifyRequest(req, []byte("{}")) {
		t.Fatal("expected request to be invalid")
	}
}
