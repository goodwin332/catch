# Catch: runbook

Дата обновления: 2026-04-17

## Локальный запуск

1. Установить зависимости:
   - Go `1.26`;
   - Node.js `24`;
   - Docker Desktop или совместимый Docker runtime.
2. Создать `.env` из `.env.example` и оставить local defaults.
3. Поднять инфраструктуру:
   - `make infra-up`
4. Применить миграции:
   - `make api-migrate`
5. Запустить API:
   - `make api-run`
6. Запустить outbox worker в отдельном терминале:
   - `make outbox-worker`
7. Запустить web:
   - `npm install`
   - `npm run web:dev`
8. Открыть `http://localhost:3000`.

## Локальная авторизация

- Email flow работает через log/outbox provider.
- Dev login доступен только вне production: кнопка на `/login`.
- OAuth-кнопки есть на `/login`, но для реального входа нужны client id/secret и redirect URL:
  - `http://localhost:3000/api/auth/oauth/google/callback`
  - `http://localhost:3000/api/auth/oauth/vk/callback`
  - `http://localhost:3000/api/auth/oauth/yandex/callback`

## Проверки перед релизом

- `cd apps/api && GOCACHE=/tmp/catch-go-build go test ./...`
- `cd apps/api && GOCACHE=/tmp/catch-go-build go vet ./...`
- `cd apps/api && TEST_DATABASE_URL='postgres://catch:catch@localhost:5432/catch_test?sslmode=disable' GOCACHE=/tmp/catch-go-build go test -tags=integration ./tests/integration -count=1`
- `ruby -e 'require "yaml"; YAML.load_file("apps/api/openapi/openapi.yaml")'`
- `npm run api:types`
- `npm run web:lint`
- `npm run web:build`
- `docker compose -f infra/compose/docker-compose.yml config`

## Production checklist

- Заменить `AUTH_SECRET`.
- Выключить `AUTH_DEV_LOGIN_ENABLED` и `AUTH_DEV_EMAIL_CODE_IN_RESPONSE`.
- Настроить SMTP или production email provider.
- Настроить Google/VK/Yandex OAuth apps.
- Настроить `STORAGE_PROVIDER=s3`, bucket policy и CDN.
- Для локальной проверки S3 использовать MinIO из `make infra-up`.
- Настроить PostgreSQL backups.
- Настроить Meilisearch master key и persistent volume.
- Запустить API, web, outbox worker и media cleanup по расписанию.
- Включить алерты по API readiness, worker lag, outbox failures, search indexing failures, storage errors.
