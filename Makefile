API_DIR := apps/api
COMPOSE_FILE := infra/compose/docker-compose.yml
DATABASE_URL ?= postgres://catch:catch@localhost:5432/catch?sslmode=disable
DATABASE_MIGRATIONS_DIR ?= apps/api/migrations

.PHONY: infra-up infra-down infra-logs api-run api-test api-test-integration api-fmt api-migrate outbox-worker media-cleanup web-dev web-build web-lint

api-run:
	cd $(API_DIR) && go run ./cmd/api

api-test:
	cd $(API_DIR) && go test ./...

api-test-integration:
	@test -n "$$TEST_DATABASE_URL" || (echo "TEST_DATABASE_URL is required for integration tests"; exit 1)
	cd $(API_DIR) && go test -tags=integration ./tests/integration

api-fmt:
	cd $(API_DIR) && gofmt -w .

api-migrate:
	DATABASE_URL="$(DATABASE_URL)" DATABASE_MIGRATIONS_DIR="$(DATABASE_MIGRATIONS_DIR)" go run ./$(API_DIR)/cmd/migrate

outbox-worker:
	DATABASE_URL="$(DATABASE_URL)" DATABASE_MIGRATIONS_DIR="$(DATABASE_MIGRATIONS_DIR)" go run ./$(API_DIR)/cmd/outbox-worker

media-cleanup:
	DATABASE_URL="$(DATABASE_URL)" DATABASE_MIGRATIONS_DIR="$(DATABASE_MIGRATIONS_DIR)" go run ./$(API_DIR)/cmd/media-cleanup --older-than=24h --limit=100

web-dev:
	npm run web:dev

web-build:
	npm run web:build

web-lint:
	npm run web:lint

infra-up:
	docker compose -f $(COMPOSE_FILE) up -d postgres meilisearch

infra-down:
	docker compose -f $(COMPOSE_FILE) down

infra-logs:
	docker compose -f $(COMPOSE_FILE) logs -f postgres meilisearch
