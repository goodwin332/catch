# Catch: план разработки

Дата обновления: 2026-04-17

## Статусы

- `[x]` сделано.
- `[~]` в работе.
- `[ ]` не начато.

## 1. Backend foundation

Статус: `[x]`

Сделано:

- Go workspace и `apps/api`.
- Конфигурация приложения и окружений.
- Structured logging.
- HTTP router, middleware, recovery, request id, security headers.
- Единый формат ошибок `application/problem+json`.
- Health/readiness endpoints.
- PostgreSQL pool.
- Transaction manager.
- Migration runner.
- Foundation migration для `users`, `user_profiles`, `auth_sessions`, `email_login_codes`, `oauth_accounts`, `outbox_events`.
- Базовые tests для config, email value object, rating capabilities.
- Локальный Docker Compose для PostgreSQL.
- Makefile-команды для infra, migrate и integration tests.
- Foundation migration проверена на локальной PostgreSQL без seed-данных.
- Global in-memory HTTP rate limiter.
- GitHub Actions API CI: gofmt, tests, vet, integration tests, OpenAPI YAML validation.
- Local `catch_test` database used for integration-test verification, without seed data in main database.
- Minimal Prometheus-compatible `/metrics` endpoint.

Осталось:
- Structured business metrics after workers and media pipeline appear.

## 2. API contracts

Статус: `[x]`

Сделано:

- OpenAPI для auth endpoints.
- Единые error codes.
- Базовые DTO для current user и capabilities.
- OpenAPI для profile, articles, feed/search, comments, reactions, bookmarks/subscriptions.
- OpenAPI для reports, moderation, notifications и chat.
- OpenAPI для media/files.
- OpenAPI для notifications SSE stream.
- Generated TypeScript client после появления frontend workspace.
- OpenAPI drift check в CI.
- Moderation thread list/read and reopen contracts.
- Bookmark list names in bookmarked article responses.

Осталось:

- Поддерживать drift check при каждом изменении контрактов.

## 3. Auth + sessions

Статус: `[x]`

Сделано:

- Auth routes scaffold.
- Таблицы под users, sessions, email codes, OAuth accounts.
- Identity repositories для users, sessions, email codes.
- Session token manager.
- Session cookies: HttpOnly session cookie и отдельный CSRF cookie.
- CSRF validation для authenticated logout.
- Общий auth middleware.
- Общий CSRF middleware для protected state-changing endpoints.
- Email code request/verify flow.
- Email code request emits `auth.email_code.requested` outbox event.
- Outbox worker delivery adapter sends email-code messages through mail sender interface.
- Mail provider factory supports disabled, log and SMTP senders.
- `/auth/me`.
- `/auth/logout`.
- Dev-only login с автоматическим созданием одного локального пользователя.
- Dev-only login supports multiple local users without username conflicts.
- Dev-only выдача email-кода в response только вне production.
- Opt-in integration test для dev login, `/auth/me` и `/auth/logout`.
- Auth integration tests подключены к API CI и проверены локально на выделенной `catch_test`.
- Rate limits для email code request, email verify и dev login.
- OAuth provider flow для Google/VK/Yandex: signed state cookie, PKCE, provider config, token/userinfo exchange, linking через `oauth_accounts`.
- OAuth integration test на fake provider без внешних секретов.

Осталось:

- Production SMTP/provider credentials and deliverability setup.
- Production OAuth app credentials and provider-specific кабинеты.

## 4. Profile

Статус: `[x]`

Сделано:

- Profile API contracts.
- `GET /profile/me`.
- `PATCH /profile/me`.
- `GET /profiles/{username}`.
- Private profile response keeps birth date private.
- Public profile response excludes email and birth date.
- PostgreSQL profile repository.
- Profile service validation.
- Username normalization rules.
- Public profile lookup supports username and fallback user id links from feed/comment surfaces.
- Frontend profile page has follow, unfollow and start-chat actions.
- Profile edit form has country/city search suggestions.

Осталось:

- Avatar media integration.
- Production country/city справочник или внешний provider.

## 5. Articles + drafts + publishing workflow

Статус: `[x]`

Сделано:

- Article API contracts для draft foundation.
- Structured JSON article document validation.
- Таблицы `articles`, `article_revisions`, `tags`, `article_revision_tags`.
- `POST /articles/drafts`.
- `GET /articles/drafts/{articleID}`.
- `PATCH /articles/drafts/{articleID}` с новой ревизией.
- `POST /articles/drafts/{articleID}/submit`.
- Rating-based routing: direct publication for rating `>= 1000`, moderation for lower rating.
- Scheduled publication validation: future date not дальше одного месяца.
- Max 10 tags validation.
- Low-rating submit создаёт `moderation_submissions`.
- `ready_to_publish` после модерации можно опубликовать без повторного требования прямого рейтинга.
- `GET /articles/my` для списка статей текущего автора.
- Media/files foundation для editor uploads: metadata, local storage, upload/content endpoints.
- Связь media files с article revisions.
- Frontend draft editor: save draft, upload media, submit draft.
- Integration test для low-rating draft -> moderation -> approval -> publication workflow.
- Integration test confirms invalid media MIME payloads are rejected.
- Image upload validates real PNG/JPEG/GIF dimensions before storage.
- Media metadata stores optional width and height for image files.
- Media storage boundary supports local storage and S3-compatible object storage via `STORAGE_PROVIDER`.
- Media cleanup application service and `cmd/media-cleanup`.
- `make media-cleanup` for one-shot local cleanup.
- Integration test confirms unreferenced media is deleted while revision-linked media remains available.
- Article document validates `geo_point.radius_meters` with 10 km maximum.
- Editor supports media preview flow, geo point, route, tag suggestions and live article preview.
- First image block is used as article cover in feed/article DTOs.

Осталось:

- Scheduled media cleanup wiring in production environment.
- Published immutable snapshot read model.
- Full Confluence-class rich editor remains post-MVP.

## 6. Feed + search

Статус: `[x]`

Сделано:

- `GET /articles/{articleID}` для опубликованных статей.
- `GET /feed` для публичной ленты опубликованных статей.
- `GET /articles/feed` для персональной ленты с приоритетом подписок.
- `GET /search?q=` для PostgreSQL fallback search.
- Поиск начинается с 3 символов.
- Public visibility filter: только `status = published` и `published_at <= now`.
- Title matches ранжируются выше body/tag matches.
- Frontend `/search` page wired to backend search with empty and short-query states.
- Integration test confirms accepted article report removes article from public read path.
- Cursor pagination for public feed, personalized feed and PostgreSQL fallback search.
- Frontend feed and search pages support opaque `next_cursor` links.
- Integration test covers feed/search cursor pagination and invalid cursor validation.
- Search indexing foundation through Meilisearch-compatible outbox handler.
- `/search` uses Meilisearch query path and hydrates current PostgreSQL article read models.
- PostgreSQL search remains fallback when external search is unavailable.
- Meilisearch index settings define searchable, displayed, filterable, sortable attributes and ranking rules.
- Local Meilisearch service in Docker Compose and env configuration.
- `@` people search via `/search/people`.
- `#` tag queries normalize to tag search in article search.
- Popular feed endpoint for the last 14 days and homepage popular tab.
- Integration tests cover search indexing/delete, people search, tag search and popular feed ranking.
- Public feed order follows newer publication day first, then higher reaction score inside the day.
- Personalized feed prioritizes followed authors from the last 3 days.
- Feed cards show relative dates, reaction ratio badges and bookmark action.
- Article pages show related articles.

Осталось:

- Search relevance tuning with real production queries and synonym dictionary.

## 7. Comments + reactions + rating

Статус: `[x]`

Сделано:

- Таблицы `comments`, `reactions`, `rating_events`.
- `GET /articles/{articleID}/comments`.
- `GET /comments/{commentID}` for comment permalink/read contract.
- `POST /articles/{articleID}/comments`.
- `PATCH /comments/{commentID}` for author edit within 1 hour.
- `POST /reactions`.
- Rating threshold for comments: `>= -100`.
- Rating threshold for reactions: `>= 0`.
- One reaction per target/user.
- Reaction changes update author rating through `rating_events`.
- Reaction response returns `reactions_up`, `reactions_down` and `reaction_score`.
- Article and comment read models expose reaction counters without denormalized tables.
- Frontend article comments: tree, reply, permalink, create and edit own comment.
- Frontend article cards and article page show reaction score.
- Integration test covers comment create/edit/permalink, reaction counters and accepted comment report deletion.
- Article page hides comment/reply entry points for low-rating users and shows rating requirement message.
- Notification text for comments uses 10-character preview helper.

Осталось:

- Policy tests for thresholds.

## 8. Bookmarks + subscriptions

Статус: `[x]`

Сделано:

- Таблицы `bookmark_lists`, `bookmark_items`, `follows`.
- Default bookmark list создаётся лениво при первом обращении к закладкам.
- `GET /bookmarks/lists`.
- `POST /bookmarks/lists`.
- `POST /bookmarks/items`.
- `GET /bookmarks/items` со списком и поиском опубликованных статей в закладках.
- `DELETE /bookmarks/items`.
- `POST /subscriptions/{authorID}`.
- `DELETE /subscriptions/{authorID}`.
- Follow/unfollow updates author rating через `rating_events`.
- Лимиты списков и элементов заложены в backend.
- Bookmark add rate limit: 20/minute.
- Bookmark page has list tabs, search, filtering, removal and list names on saved articles.
- My articles page has status tabs, counts and local search.
- Profile/follow flows are linked from public profiles.

Осталось:

- Drag reorder bookmark lists.

## 9. Notifications

Статус: `[~]`

Сделано:

- Таблица `notifications`.
- Индексы для unread/grouped notifications.
- `GET /notifications`.
- `GET /notifications/unread-count`.
- `POST /notifications/{notificationID}/read`.
- `POST /notifications/read-target`.
- User-scoped access: пользователь видит и изменяет только свои уведомления.
- Shared notification/outbox helper.
- Notification side effects для comments, moderation approvals/rejections, accepted reports и chat messages.
- Outbox worker command для обработки pending events и retries.
- `GET /notifications/stream` как SSE delivery foundation для unread counters.
- Frontend notification center and unread badge.
- Web-layer SSE proxy для browser-side EventSource без прямого CORS/cookie доступа к backend.
- Integration test на `GET /notifications/stream`, включая совместимость с middleware.
- Same-origin polling fallback for unread notification count when SSE fails.
- Notification grouping migration with unread duplicate merge and unique partial index.
- Shared notification helper increments unread grouped notifications.
- Integration test confirms grouped chat notifications.
- Notification side effects для follow, bookmark, rating и article publication.
- Article publication emits outbox event; outbox worker fans notifications out to current followers.
- Integration test covers social notification producers.
- Notification text preview helper truncates inserted text to 10 characters plus ellipsis.

Осталось:

- Delivery adapters for email/push channels after product channel decisions.

## 10. Moderation

Статус: `[x]`

Сделано:

- Таблицы `moderation_submissions`, `moderation_approvals`, `moderation_threads`.
- State machine statuses для submissions и threads.
- `GET /moderation/submissions`.
- `POST /moderation/submissions/{submissionID}/approve`.
- `POST /moderation/submissions/{submissionID}/reject`.
- `POST /moderation/submissions/{submissionID}/threads`.
- `GET /moderation/submissions/{submissionID}/threads`.
- `POST /moderation/threads/{threadID}/resolve`.
- `POST /moderation/threads/{threadID}/reopen`.
- Approval threshold: 5 moderators or admin approval.
- Admin-only rejection with required reason.
- Approved article goes to `ready_to_publish`.
- Rejected article goes to `archived`.
- Submission response includes approval and open-thread counters.
- Frontend moderation has separate tabs for articles, article reports and comment reports.
- Frontend moderation shows threads, resolve/reopen actions and direct links to target entities.
- Integration test на approval workflow через HTTP + session/CSRF.

Осталось:

- Author replies inside moderation threads.

## 11. Complaints/reports

Статус: `[x]`

Сделано:

- Таблицы `reports`, `report_decisions`.
- Reason validation на уровне БД.
- `other` requires details на уровне БД.
- `POST /reports`.
- `GET /reports` для moderation queue.
- `POST /reports/{reportID}/decisions`.
- Порог создания жалобы: rating `>= 10`.
- Порог решения жалоб: rating `>= 10000` или admin.
- Decision thresholds: comments `3 accept / 5 reject`, articles `5 accept / 10 reject`, admin override.
- Accepted article report removes article from public feed/search.
- Accepted comment report marks comment deleted.
- Accepted report applies rating penalty through ledger.
- Report creation rate limit: 1 report per 5 minutes.
- Integration test на article report creation and admin accept workflow.
- Integration test на comment report creation and admin accept workflow.

Осталось:

- Больше edge-case тестов для duplicate reports and non-admin thresholds.

## 12. Chat/realtime

Статус: `[~]`

Сделано:

- Таблицы `chat_conversations`, `chat_conversation_members`, `chat_messages`.
- Message sent/read statuses.
- Conversation domain/service/repository/http layers.
- `POST /chat/conversations`.
- `GET /chat/conversations`.
- `GET /chat/conversations/{conversationID}/messages`.
- `POST /chat/conversations/{conversationID}/messages`.
- `POST /chat/conversations/{conversationID}/read`.
- User-scoped access: читать и писать можно только в своих диалогах.
- Chat permission threshold: rating `>= -100` или admin.
- Chat message rate limit: 20 messages/minute.
- Frontend chat uses a two-pane conversation layout with SSE updates and read action.
- Integration test на direct conversation, message send/list and mark-read workflow.
- Chat notification text uses 10-character message preview.

Осталось:

- Production WebSocket gateway and presence/typing indicators.

## 13. Hardening

Статус: `[~]`

Сделано:

- Global in-memory HTTP rate limiter на уровне router.
- Per-action rate limits для email auth, dev login, bookmarks, reports и chat messages.
- Audit log migration для state-changing API requests.
- Audit middleware для `POST/PUT/PATCH/DELETE` внутри `/api/v1`.
- Streaming-compatible middleware: `http.Flusher` preserved through status recorders, timeout skipped for `/stream` endpoints.
- Integration cleanup now resets `users`, `audit_log` and `outbox_events` without leaking cross-test state.
- Migration `000006_audit_log` применена на локальной PostgreSQL.
- CI checks для gofmt, tests, integration tests, vet и OpenAPI YAML.
- Docker Compose configuration check for PostgreSQL, MinIO and Meilisearch.

Осталось:

- Security headers review.
- Extended file validation: dimensions, antivirus/provider hook, filename policy.
- Outbox idempotency tests.
- Backup/restore checks.

## 14. Frontend foundation

Статус: `[~]`

Сделано:

- Next.js workspace `apps/web`.
- React 19 / Next.js 16 SSR-first foundation.
- Root npm workspace.
- Design tokens перенесены из `design/uikit.html`: Inter, nature palette, slate surfaces, dark variables.
- SSR homepage с публичной лентой через `/feed` и fallback empty-state data.
- Базовый API client для frontend/backend контракта.
- Web CI: install, typecheck, build.
- Generated TypeScript client from OpenAPI через `openapi-typescript`.
- CI drift check для generated API types.
- Frontend auth shell: SSR `/auth/me`, `/login`, dev login server action, logout server action.
- Frontend route shells: article page, public profile, bookmarks, moderation queue, chat conversations.
- Dark theme toggle and mobile header navigation.
- Article draft creation shell wired to backend `POST /articles/drafts`.
- Article edit page wired to draft read/update/media upload/submit.
- Article editor supports provisional `geo_point` and `route` document blocks.
- Article editor keeps existing media blocks on save and shows live preview for text, media, geo point and route.
- Public article page renders structured article document instead of raw JSON.
- Frontend actions for bookmarks, reports, moderation decisions and chat messages.
- Notifications page and unread badge wired to backend.
- Browser-side unread badge updates through same-origin SSE proxy.
- Browser-side unread badge falls back to same-origin polling.
- SSR search page wired to backend `/search`.
- Chat messages support polling-friendly `after_id`.
- Chat messages SSE stream endpoint is available for realtime MVP.
- Chat page consumes same-origin SSE proxy for realtime message delivery.
- Login page includes email, dev-only and OAuth provider entry points through SSR-safe proxy routes.
- Global loading, error and not-found states are implemented.
- Header navigation has active states.
- Home feed has public, popular and subscriptions tabs.
- My articles page lists draft, moderation, scheduled, published and archived states with tabs, counts and search.
- Profile edit page is wired to `/profile/me`.
- Bookmarks page supports list tabs, search, filtering and removal.
- Article page supports reactions, report panel, bookmark list picker and threaded comments.
- Moderation page has article/report tabs, counters, thread creation, resolve/reopen and decision actions.
- Notifications can open target entities and mark all notifications for that target as read.
- Article cards show relative date, reaction ratio and bookmark action.
- Editor includes media upload zone, cover preview rule, tag suggestions, geo radius guard and live preview.

Осталось:

- Browser QA on real target devices before public launch.
- Production provider credentials and deployment secrets.

## Ближайший порядок

1. Провести product QA по core flows на реальных браузерных сценариях.
2. Настроить production secrets, domains, CDN and provider credentials.
3. Подключить scheduled jobs in production environment.

## Последняя проверка

Дата: 2026-04-17

- `cd apps/api && go test ./...` — passed.
- `cd apps/api && go vet ./...` — passed.
- `npm run api:types` — passed; OpenAPI parsed and generated TypeScript types.
- `go test -tags=integration ./tests/integration` на `catch_test` — passed.
- `npm run web:lint` — passed.
- `npm run web:build` — passed.
- `docker compose -f infra/compose/docker-compose.yml config` — passed.
