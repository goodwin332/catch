# Catch: итоговое резюме

Дата: 2026-04-17

## Что изучено

- `REQ.md`: требования к продукту, рейтингу, правам, статьям, модерации, жалобам, ленте, поиску, чату, уведомлениям, закладкам и лимитам.
- `design/*.html`: UI kit, лента, статья, редактор, модерация, чат, профиль.
- Структура репозитория: реализованы Go backend, Next.js web layer, OpenAPI contracts, миграции, docker compose и документация.

Отсутствующие входные данные:

- отдельные папки `uikit/`, `mockups/`, `assets/`, `assets/ui`;
- финальный бренд Catch: логотип, favicon, production assets;
- требования к провайдерам карт, email, CDN, deployment target.

## Product core

Catch - русскоязычная Habr-like content platform про активный отдых, рыбалку и охоту.

Ключевое ядро:

- публичная SSR-лента статей;
- SSR-страница статьи;
- структурированный редактор статей;
- комментарии;
- реакции и рейтинг;
- права, завязанные на рейтинг;
- pre-publish moderation;
- complaints/reports workflow;
- поиск;
- закладки;
- подписки;
- уведомления;
- чат;
- профиль.

## Выбранная архитектура

Выбрана гибридная архитектура:

- backend core: Go modular monolith;
- frontend: Next.js SSR-first web layer;
- worker: Go background jobs;
- database: PostgreSQL 18;
- search: PostgreSQL FTS + Meilisearch;
- realtime: outbox + Centrifugo/WebSocket target, SSE/polling fallback;
- storage: S3-compatible object storage;
- repository model: monorepo.

## Recommended stack

- Go 1.26.
- Next.js 16.
- React 19.
- TypeScript.
- CSS design tokens from project UI kit.
- PostgreSQL 18.
- Meilisearch.
- Redis/Valkey-compatible cache/rate limit store.
- Centrifugo for production realtime.
- Local filesystem storage for development, S3-compatible object storage for production, MinIO locally for S3 checks.
- OpenAPI contracts.
- OpenTelemetry, structured JSON logs, Prometheus metrics.

## Почему выбран этот стек

- Catch требует SEO и SSR для ленты, статей, профилей, тегов и поиска.
- Редактор, модерация, чат и уведомления требуют rich frontend runtime.
- Рейтинг, модерация, жалобы и публикация требуют транзакционной backend-модели.
- Go подходит для domain core, outbox, workers, rate limits, search indexing и realtime publishing.
- PostgreSQL подходит для реляционной модели, транзакций, constraints, JSONB article document и fallback search.
- Meilisearch дает качественный публичный поиск без операционной цены OpenSearch.
- Monorepo снижает риск рассинхронизации API, UI и доменных контрактов.

## Отклонённые альтернативы

- Монолитный SSR отклонен как основной вариант: быстрее для CRUD/auth/admin, но хуже для rich editor, block-level moderation, chat и сложной дизайн-системы.
- Полностью раздельные backend/frontend репозитории отклонены: на текущей стадии добавляют лишнюю координацию и риск contract drift.
- SPA-only отклонен: конфликтует с SEO/SSR и ухудшает безопасность session/auth.
- PHP/Laravel отклонен как основной стек: быстрее для MVP, но выше риск framework-centric домена и хуже долгосрочный контроль realtime/workers.
- Python/Django отклонен как основной стек: зрелый вариант для admin/CRUD, но Go лучше подходит для workers/realtime/rate limits/outbox.
- OpenSearch отклонен для MVP: слишком дорогой в эксплуатации для текущих требований.
- PostgreSQL-only search отклонен как финальный публичный поиск: подходит как fallback, но слабее по typo tolerance и search-as-you-type.

## MVP

MVP включает:

- foundation monorepo;
- auth + profile;
- articles + drafts + publish workflow;
- SSR feed + SSR article page;
- basic search;
- comments + reactions + rating ledger;
- bookmarks + subscriptions;
- notifications;
- moderation;
- complaints/reports;
- hardening;
- release readiness.

Chat/realtime входит в MVP только при продуктовой необходимости приватных сообщений на запуске. Иначе он идет после стабилизации core flows.

## Post-MVP

После стабилизации core flows реализуются:

- full Centrifugo/WebSocket realtime gateway;
- typing indicators and presence;
- advanced search suggestions and relevance tuning;
- bookmark drag reorder;
- rich map/route editing and GPX import/export;
- public collections;
- ML moderation assist;
- recommendations;
- mobile app/public API;
- domain microservices.

## Главные инженерные риски

- Неверная модель статьи. Решение: structured article document with stable block ids.
- Рейтинг без ledger. Решение: append-only rating events plus aggregate.
- Права без policy layer. Решение: centralized backend policies.
- Модерация без revision model. Решение: moderation submission per article revision.
- Search без visibility discipline. Решение: public index from outbox only.
- Realtime без persisted state. Решение: commit first, publish after outbox.
- Media без lifecycle. Решение: temporary, attached_to_draft, published, orphaned, deleted.
