# Catch: финальная структура проекта

Дата: 2026-04-15

## Принцип

Catch реализуется как monorepo с тремя runtime: `web`, `api`, `worker`. Backend остается modular monolith и источником истины для домена, прав, рейтинга, модерации и транзакций. Frontend отвечает за SSR, UI и интерактивные сценарии. Worker выполняет отложенные side effects через outbox.

## Tree

```text
.
├── apps
│   ├── api
│   │   ├── cmd
│   │   │   ├── api
│   │   │   └── migrate
│   │   ├── internal
│   │   │   ├── app
│   │   │   │   ├── bootstrap
│   │   │   │   ├── config
│   │   │   │   └── composition
│   │   │   ├── platform
│   │   │   │   ├── auth
│   │   │   │   ├── clock
│   │   │   │   ├── db
│   │   │   │   ├── http
│   │   │   │   ├── logger
│   │   │   │   ├── mailer
│   │   │   │   ├── metrics
│   │   │   │   ├── outbox
│   │   │   │   ├── ratelimit
│   │   │   │   ├── realtime
│   │   │   │   ├── search
│   │   │   │   ├── storage
│   │   │   │   └── tracing
│   │   │   ├── modules
│   │   │   │   ├── identity
│   │   │   │   ├── access
│   │   │   │   ├── profiles
│   │   │   │   ├── reputation
│   │   │   │   ├── articles
│   │   │   │   ├── moderation
│   │   │   │   ├── comments
│   │   │   │   ├── reactions
│   │   │   │   ├── reports
│   │   │   │   ├── feed
│   │   │   │   ├── search
│   │   │   │   ├── bookmarks
│   │   │   │   ├── notifications
│   │   │   │   ├── chat
│   │   │   │   ├── media
│   │   │   │   └── abuse
│   │   │   └── testutil
│   │   ├── migrations
│   │   ├── seeds
│   │   ├── fixtures
│   │   ├── openapi
│   │   ├── queries
│   │   └── tests
│   │       ├── integration
│   │       └── contract
│   ├── worker
│   │   ├── cmd
│   │   ├── internal
│   │   │   ├── app
│   │   │   ├── jobs
│   │   │   │   ├── outbox
│   │   │   │   ├── search_indexing
│   │   │   │   ├── notifications
│   │   │   │   ├── scheduled_publication
│   │   │   │   ├── media_processing
│   │   │   │   ├── cleanup
│   │   │   │   └── reconciliation
│   │   │   └── testutil
│   │   └── tests
│   └── web
│       ├── app
│       │   ├── (public)
│       │   ├── (auth)
│       │   ├── (moderation)
│       │   ├── layout.tsx
│       │   ├── error.tsx
│       │   └── not-found.tsx
│       ├── features
│       │   ├── auth
│       │   ├── profile
│       │   ├── articles
│       │   ├── article-editor
│       │   ├── feed
│       │   ├── comments
│       │   ├── moderation
│       │   ├── reports
│       │   ├── search
│       │   ├── bookmarks
│       │   ├── notifications
│       │   ├── chat
│       │   └── layout
│       ├── components
│       │   ├── primitives
│       │   ├── navigation
│       │   ├── feedback
│       │   └── seo
│       ├── lib
│       │   ├── api
│       │   ├── auth
│       │   ├── config
│       │   ├── format
│       │   ├── telemetry
│       │   └── validation
│       ├── styles
│       ├── public
│       └── tests
├── packages
│   ├── contracts
│   ├── ui
│   ├── article-document
│   ├── config
│   ├── eslint-config
│   └── tsconfig
├── infra
│   ├── compose
│   ├── docker
│   ├── nginx
│   ├── postgres
│   ├── meilisearch
│   ├── centrifugo
│   ├── object-storage
│   ├── observability
│   └── deploy
├── docs
│   ├── project-structure.md
│   ├── requirements.md
│   └── summary.md
├── design
├── scripts
├── tools
├── .env.example
├── README.md
└── Makefile
```

## Ключевые папки

- `apps/api`: Go backend API. Здесь находятся доменные модули, handlers, use cases, repositories, OpenAPI, migrations и backend-тесты.
- `apps/worker`: Go worker. Обрабатывает outbox, индексацию поиска, уведомления, отложенную публикацию, media processing, cleanup и reconciliation.
- `apps/web`: Next.js SSR-first frontend. Публичные страницы, личные кабинеты, модерация, редактор, чат, уведомления и UI composition.
- `packages/contracts`: OpenAPI, generated TypeScript types, общие коды ошибок.
- `packages/ui`: дизайн-токены и UI primitives. Не содержит доменную бизнес-логику.
- `packages/article-document`: схема структурированной статьи, validation, renderer primitives.
- `infra`: docker/compose/deploy/observability конфигурации.
- `docs`: только финальные документы проекта.
- `design`: входные HTML-макеты и UI kit.

## Backend module layout

Каждый модуль в `apps/api/internal/modules/<module>` организуется одинаково:

```text
<module>
├── domain
├── app
│   ├── commands
│   ├── queries
│   └── dto
├── ports
├── adapters
│   ├── http
│   ├── postgres
│   └── external
└── tests
```

- `domain`: entities, value objects, errors, state machines, domain events.
- `app`: use cases, command/query handlers, transaction orchestration.
- `ports`: repository and external dependency interfaces.
- `adapters/http`: handlers/controllers.
- `adapters/postgres`: repository implementations.
- `adapters/external`: clients for providers, search, storage, realtime, mail.

## Domain modules

- `identity`: users, emails, OAuth accounts, sessions, email codes.
- `access`: roles, sanctions, rating-based policies.
- `profiles`: public/private profile fields, avatar, location, boat.
- `reputation`: rating ledger and aggregate rating.
- `articles`: drafts, revisions, published snapshots, tags, article document.
- `moderation`: article review queue, block threads, approvals, rejections.
- `comments`: tree comments, edit window, soft delete, permalinks.
- `reactions`: article/comment likes and dislikes.
- `reports`: complaints workflow for articles and comments.
- `feed`: fresh/popular/subscriptions feed projections.
- `search`: query routing, PostgreSQL fallback, external search adapter.
- `bookmarks`: bookmark lists, membership, ordering, limits.
- `notifications`: grouped notifications, unread counters, routing.
- `chat`: conversations, messages, delivery/read receipts.
- `media`: uploads, validation, previews, lifecycle.
- `abuse`: rate limits, duplicate prevention, spam signals.

## Code organization rules

- Backend is the source of truth for permissions, rating, moderation state, report state and rate limits.
- Handlers parse request, call use cases and map responses. They do not contain business logic.
- Use cases own transactions and policy checks.
- Repositories do not decide permissions.
- Rating is updated only through `reputation` ledger.
- Article content is stored as structured document with stable block ids.
- Search, notifications and realtime are side effects emitted through outbox.
- Worker jobs are idempotent.
- Frontend receives capabilities from backend and never computes final permissions locally.
- Shared code belongs in `packages/*` only when it is domain-neutral or explicitly shared schema, such as article document.
- Domain-specific logic stays in the owning backend module or frontend feature.

## Config, migrations, tests, observability

- `.env.example` documents all required variables. Secrets are never committed.
- `apps/api/migrations` contains reversible schema migrations where practical.
- `apps/api/seeds` contains deterministic dev/reference seeds.
- `apps/api/fixtures` and `apps/web/tests/fixtures` contain test data.
- CI runs backend tests, frontend typecheck/tests, OpenAPI drift check, migration check and build.
- Observability lives in `infra/observability` and platform packages: logs, metrics, tracing, outbox lag, worker failures, search lag, auth failures, policy denials.

## Naming

- Go packages: lowercase domain names, plural for modules.
- Go files: `snake_case.go`.
- TypeScript components: `PascalCase.tsx`.
- Hooks: `useSomething.ts`.
- SQL tables and columns: `snake_case`, plural table names.
- Events: dot notation, past tense, for example `article.published`, `report.accepted`.
- Code comments, when needed, are in Russian.

## Forbidden patterns

- SPA-only public pages.
- Rating as mutable number without ledger.
- Permission checks only in frontend.
- Raw HTML as the only article source.
- Search indexing of drafts, private data or removed content.
- Realtime publish before DB commit.
- Worker as source of business decisions.
- Early microservices for rating, moderation, reports, comments, notifications or chat.
- Domain logic in `platform` or `packages/ui`.

