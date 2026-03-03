# middleman

## Local Run

Required environment variables:

- `DATABASE_URL`
- `TELEGRAM_TOKEN`
- `TELEGRAM_WEBHOOK_SECRET`
- `TELEGRAM_WEBHOOK_URL` (optional: if set, app calls Telegram `setWebhook` on startup)
- `DISCORD_TOKEN`
- `SLACK_TOKEN`
- `SLACK_SIGNING_SECRET`
- `HTTP_PORT` (optional, default `8080`)
- `HTTP_READ_TIMEOUT_SEC` (optional, default `10`)
- `HTTP_WRITE_TIMEOUT_SEC` (optional, default `20`)
- `HTTP_IDLE_TIMEOUT_SEC` (optional, default `60`)
- `DELIVERY_ALERT_INTERVAL_SEC` (optional, default `30`)
- `DELIVERY_ALERT_RETRY_WINDOW_SEC` (optional, default `300`)
- `DELIVERY_ALERT_FAILED_THRESHOLD` (optional, default `10`)
- `DELIVERY_ALERT_BACKLOG_THRESHOLD` (optional, default `100`)
- `DELIVERY_ALERT_RETRY_SPIKE_THRESHOLD` (optional, default `30`)

Health endpoints:

- `GET /health` (liveness)
- `GET /ready` (readiness + delivery queue snapshot)

## Tests

Run unit tests:

```bash
make test
```

Run integration tests for PostgreSQL repositories:

```bash
make test-integration
```
