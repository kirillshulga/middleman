COMPOSE_FILE := deployments/docker-compose.yml
TEST_DATABASE_URL ?= postgres://sync_user:sync_pass@localhost:5432/sync_db?sslmode=disable

.PHONY: run test test-integration vet release-check db-up db-down db-reset migrate

run:
	go run ./cmd

test:
	go test ./...

test-integration:
	TEST_DATABASE_URL='$(TEST_DATABASE_URL)' go test ./internal/repository/postgres -run Integration -v

vet:
	go vet ./...

release-check:
	go test ./... && go vet ./...

db-up:
	docker compose -f $(COMPOSE_FILE) up -d postgres

db-down:
	docker compose -f $(COMPOSE_FILE) down

db-reset:
	docker compose -f $(COMPOSE_FILE) down -v
	docker compose -f $(COMPOSE_FILE) up -d postgres
	cat migrations/001_init.sql | docker exec -i sync-postgres psql -U sync_user -d sync_db

migrate:
	cat migrations/001_init.sql | docker exec -i sync-postgres psql -U sync_user -d sync_db
