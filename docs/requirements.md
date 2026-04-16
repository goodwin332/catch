# Catch: финальные требования к реализации

Дата: 2026-04-15

## Входные ограничения

- Основной файл требований: `REQ.md`.
- Дизайн-источник: `design/uikit.html` и HTML-макеты в `design/`.
- Отдельные папки `uikit/`, `mockups/`, `assets/`, `assets/ui` отсутствуют.
- Исходного приложения, схем БД, API и тестов нет.

## Технологические требования

- Backend: Go 1.26.
- Frontend: Next.js 16, React 19, TypeScript.
- Styling: Tailwind with local build and design tokens.
- Primary database: PostgreSQL 18.
- Search: Meilisearch for public search, PostgreSQL FTS for fallback and scoped private searches.
- Realtime: Centrifugo/WebSocket target, SSE/polling fallback.
- Jobs: Go worker with transactional outbox.
- Storage: S3-compatible object storage; MinIO for local development.
- Contracts: OpenAPI with generated TypeScript client.
- Observability: OpenTelemetry, structured JSON logs, Prometheus metrics.
- Repository: monorepo.

## Architecture requirements

- Backend is the source of truth.
- Frontend never makes final permission decisions.
- Domain modules are isolated inside `apps/api/internal/modules`.
- Every protected action uses backend policy checks.
- All side effects use outbox: search indexing, notifications, realtime, email, media processing.
- Worker jobs are idempotent.
- Public pages are SSR-first.
- Article content is stored as structured document, not raw HTML-only.
- Published article snapshot is immutable.
- Draft/revision/published states are separate.
- Moderation and reports use explicit state machines.

## Product requirements

### Auth and profile

- Email registration with code.
- Login/logout with server-side sessions.
- Google, VK and Yandex OAuth support.
- Profile fields: name, username, email, birth date, avatar, bio, boat, country, city.
- Birth date is private.
- Username and email are unique.

### Rating and access

- Default rating is `0`.
- Rating maximum is `1000000`.
- Rating is stored as ledger plus aggregate.
- Thresholds:
  - rating `< 0`: no article creation;
  - rating `< -100`: no comments and chat messages;
  - rating `< 10`: no reports;
  - rating `< 1000`: articles require moderation;
  - rating `>= 1000`: direct publication;
  - rating `>= 10000`: article and report moderation;
  - rating `> 100000`: direct chat with development lead.
- Static roles support admin override.
- Sanctions override rating.

### Articles

- Article has title, structured content, images, files, geo points, routes and tags.
- Max tags per article: 10.
- Draft is default state.
- Publication is explicit.
- Scheduled publication accepts only future date within one month.
- Published article is not edited directly.
- Low-rating articles go through moderation.

### Moderation

- Moderation access: rating `>= 10000` or admin.
- 5 moderator approvals or 1 admin approval are required.
- Threads attach to article revision and stable block id.
- Author can reply to threads but cannot create review threads.
- Non-author moderator can resolve thread.
- Author edit resets approvals.
- Rejection requires reason.

### Comments and reactions

- Comments require rating `>= -100`.
- Comments have no Markdown, no files, no images; links are allowed.
- Comments are nested.
- Comment edit window is 1 hour.
- Comment permalink opens article at target comment.
- Reactions require rating `>= 0`.
- One user has one reaction per target.
- Reactions update rating through ledger.

### Reports

- Reports require rating `>= 10`.
- Reasons: advertising, profanity, insult, fraud, other.
- `other` requires text.
- Report decisions require moderator rating `>= 10000` or admin.
- Accepted article report removes article from feed/search and direct link shows removed state.
- Accepted comment report shows deleted placeholder.
- Accepted report applies rating penalty once.

### Feed and search

- Guest and authenticated users can read feed.
- Feed loads by 10 articles.
- Guest feed: newer days first, within day higher likes first.
- Authenticated feed prioritizes followed authors from last 3 days.
- Search starts from 3 characters.
- Title matches rank above body matches.
- `@` searches people.
- `#` searches tags.
- Removed, draft and moderation content never appears in public search.

### Bookmarks and subscriptions

- Authenticated users have default bookmark list named `Избранное`.
- Max bookmark lists: 20.
- Max articles per list: 100.
- Bookmark add limit: 20/minute.
- Users with rating `>= -100` can follow authors.
- Follow/unfollow updates author rating through ledger.

### Notifications

- Notifications are grouped by event target.
- Unread counter is shown in navbar.
- Inserted text is full if shorter than 10 characters; otherwise truncated to 10 characters plus ellipsis.
- Opening target resolves the relevant grouped notification.

### Chat

- Chat requires rating `>= -100`.
- Messages support text and links.
- Messages have sent/read states.
- Message limit: 20/minute.
- Realtime delivery uses persisted state first, event publish after commit.

## Design requirements

Design source is `design/uikit.html` plus page mockups in `design/`.

### Tokens

- Font: Inter.
- Primary accent: `nature`.
- Light page background: `slate-50`.
- Light surface: `white`.
- Dark page background: `#0f172a`.
- Dark surface: `#1e293b`.
- Destructive: rose.
- Warning/pending: amber.
- Neutral: slate.

### UI rules

- Use local Tailwind build, not CDN.
- Use design tokens, not hardcoded colors in feature components.
- Keep `darkMode: class`.
- Default theme follows OS preference.
- User theme choice overrides OS.
- Major panels use `rounded-2xl`.
- Inputs/buttons use `rounded-xl`.
- Small icon controls use `rounded-lg`.
- Public page gutters: `px-4 sm:px-6 lg:px-8`.
- Main content max width: `max-w-7xl`.
- Editor max width: `max-w-4xl`.
- Article body uses readable typography with relaxed line-height.

### Components

Required primitives:

- Button;
- IconButton;
- Input;
- Textarea;
- Combobox;
- Modal;
- DropdownMenu;
- Tabs;
- Badge;
- Card;
- Avatar;
- Tooltip;
- Skeleton;
- EmptyState;
- Alert;
- FormField.

Required domain components:

- ArticleCard;
- ArticleRenderer;
- ArticleEditor;
- CommentTree;
- VoteControl;
- ReportModal;
- BookmarkPicker;
- NotificationDropdown;
- ChatConversation;
- ModerationThread.

### States

Every async or interactive surface includes:

- default;
- hover;
- focus-visible;
- disabled;
- loading;
- empty;
- error;
- success where applicable.

Dark theme is required for every component.

### Accessibility

- Semantic HTML is required.
- Icon-only buttons require accessible names.
- Modals trap focus and restore focus.
- Dropdowns support keyboard navigation.
- Forms connect labels, help text and errors.
- Color is never the only state signal.
- Reduced motion is respected.
- Long Russian text wraps safely on mobile.

## Engineering rules

- Use cases own transactions.
- Repositories do not enforce permissions.
- Handlers stay thin.
- API contracts are generated and checked for drift.
- Migrations are reviewed with code.
- Tests cover policy thresholds, state machines, ledger idempotency, visibility, rate limits and worker idempotency.
- Logs do not contain passwords, email codes, OAuth tokens or secrets.
- Code comments are Russian only when comments are needed.

## Performance requirements

- SSR first response for public feed and article pages.
- Cursor pagination for feed.
- Public search uses indexed read model.
- Feed, article, search and comments have loading and error states.
- Worker lag, outbox lag and search indexing lag are observable.

## Security requirements

- Server-side sessions with `HttpOnly`, `Secure`, SameSite cookies.
- CSRF protection for state-changing browser requests.
- OAuth uses state and PKCE where supported.
- File uploads validate extension, MIME, signature and size.
- Draft/private media is not public.
- Rate limits are enforced server-side.
- Admin, moderation, reports, sanctions and rating-sensitive changes are audited.

