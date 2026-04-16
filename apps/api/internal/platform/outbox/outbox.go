package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"catch/apps/api/internal/platform/events"
	"catch/apps/api/internal/platform/mail"
	"catch/apps/api/internal/platform/search"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ID            int64
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       json.RawMessage
	Attempts      int
}

type Handler interface {
	Handle(context.Context, Event) error
}

type Worker struct {
	db         *pgxpool.Pool
	handler    Handler
	log        *slog.Logger
	workerID   string
	batchSize  int
	interval   time.Duration
	maxRetries int
}

func NewWorker(db *pgxpool.Pool, handler Handler, log *slog.Logger, workerID string) *Worker {
	return &Worker{
		db:         db,
		handler:    handler,
		log:        log,
		workerID:   workerID,
		batchSize:  25,
		interval:   time.Second,
		maxRetries: 5,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		processed, err := w.ProcessOnce(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			w.log.ErrorContext(ctx, "outbox_process_failed", slog.String("error", err.Error()))
		}
		if processed > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Worker) ProcessOnce(ctx context.Context) (int, error) {
	events, err := w.claim(ctx)
	if err != nil {
		return 0, err
	}
	for _, event := range events {
		if err := w.handler.Handle(ctx, event); err != nil {
			if markErr := w.markFailed(ctx, event, err); markErr != nil {
				return len(events), errors.Join(err, markErr)
			}
			continue
		}
		if err := w.markProcessed(ctx, event.ID); err != nil {
			return len(events), err
		}
	}
	return len(events), nil
}

func (w *Worker) claim(ctx context.Context) ([]Event, error) {
	rows, err := w.db.Query(ctx, `
		update outbox_events
		set status = 'processing',
			locked_at = now(),
			locked_by = $1,
			attempts = attempts + 1
		where id in (
			select id
			from outbox_events
			where status = 'pending'
				and available_at <= now()
			order by available_at asc, id asc
			limit $2
			for update skip locked
		)
		returning id, aggregate_type, aggregate_id, event_type, payload, attempts
	`, w.workerID, w.batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]Event, 0)
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.AggregateType, &event.AggregateID, &event.EventType, &event.Payload, &event.Attempts); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (w *Worker) markProcessed(ctx context.Context, id int64) error {
	_, err := w.db.Exec(ctx, `
		update outbox_events
		set status = 'processed',
			processed_at = now(),
			locked_at = null,
			locked_by = null,
			last_error = null
		where id = $1
	`, id)
	return err
}

func (w *Worker) markFailed(ctx context.Context, event Event, handleErr error) error {
	nextStatus := "pending"
	if event.Attempts >= w.maxRetries {
		nextStatus = "failed"
	}
	delay := time.Duration(event.Attempts*event.Attempts) * time.Minute
	_, err := w.db.Exec(ctx, `
		update outbox_events
		set status = $2,
			available_at = now() + $3::interval,
			locked_at = null,
			locked_by = null,
			last_error = $4
		where id = $1
	`, event.ID, nextStatus, fmt.Sprintf("%f seconds", delay.Seconds()), handleErr.Error())
	return err
}

type NotificationHandler struct {
	db            *pgxpool.Pool
	articleSearch search.ArticleIndexer
	mailSender    mail.Sender
	log           *slog.Logger
}

func NewNotificationHandler(db *pgxpool.Pool, articleSearch search.ArticleIndexer, mailSender mail.Sender, log *slog.Logger) *NotificationHandler {
	if articleSearch == nil {
		articleSearch = search.NoopArticleIndexer{}
	}
	if mailSender == nil {
		mailSender = mail.NoopSender{}
	}
	return &NotificationHandler{db: db, articleSearch: articleSearch, mailSender: mailSender, log: log}
}

func (h *NotificationHandler) Handle(ctx context.Context, event Event) error {
	switch event.EventType {
	case "article.published":
		return h.handleArticlePublished(ctx, event)
	case "report.accepted":
		return h.handleReportAccepted(ctx, event)
	case "auth.email_code.requested":
		return h.handleEmailCodeRequested(ctx, event)
	case "notification.created":
		h.log.InfoContext(ctx, "notification_delivery_ready", slog.Int64("outbox_id", event.ID), slog.String("aggregate_id", event.AggregateID))
	default:
		h.log.InfoContext(ctx, "outbox_event_acknowledged", slog.Int64("outbox_id", event.ID), slog.String("event_type", event.EventType))
	}
	return nil
}

type emailCodeRequestedPayload struct {
	Email   string `json:"email"`
	Code    string `json:"code"`
	Purpose string `json:"purpose"`
}

func (h *NotificationHandler) handleEmailCodeRequested(ctx context.Context, event Event) error {
	var payload emailCodeRequestedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}
	if payload.Email == "" || payload.Code == "" {
		return nil
	}
	subject := "Код входа в Catch"
	text := "Ваш код для входа в Catch: " + payload.Code
	if payload.Purpose == "registration" {
		subject = "Код регистрации в Catch"
		text = "Ваш код для регистрации в Catch: " + payload.Code
	}
	return h.mailSender.Send(ctx, mail.Message{To: payload.Email, Subject: subject, Text: text})
}

type articlePublishedPayload struct {
	ArticleID string `json:"article_id"`
	AuthorID  string `json:"author_id"`
	Title     string `json:"title"`
}

func (h *NotificationHandler) handleArticlePublished(ctx context.Context, event Event) error {
	var payload articlePublishedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}
	if payload.ArticleID == "" || payload.AuthorID == "" {
		return nil
	}

	if err := h.indexPublishedArticle(ctx, payload.ArticleID); err != nil {
		return err
	}

	rows, err := h.db.Query(ctx, `
		select follower_id::text
		from follows
		where author_id = $1
	`, payload.AuthorID)
	if err != nil {
		return err
	}
	defer rows.Close()

	followerIDs := make([]string, 0)
	for rows.Next() {
		var followerID string
		if err := rows.Scan(&followerID); err != nil {
			return err
		}
		followerIDs = append(followerIDs, followerID)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, followerID := range followerIDs {
		if err := events.Notify(ctx, h.db, events.NotificationInput{
			UserID:     followerID,
			EventType:  "article.published",
			TargetType: "article",
			TargetID:   payload.ArticleID,
			Title:      "Новая статья",
			Body:       payload.Title,
		}); err != nil {
			return err
		}
	}
	h.log.InfoContext(ctx, "article_publication_fanout_ready", slog.Int64("outbox_id", event.ID), slog.Int("followers", len(followerIDs)))
	return nil
}

type reportAcceptedPayload struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
}

func (h *NotificationHandler) handleReportAccepted(ctx context.Context, event Event) error {
	var payload reportAcceptedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}
	if payload.TargetType != "article" || payload.TargetID == "" {
		h.log.InfoContext(ctx, "outbox_event_acknowledged", slog.Int64("outbox_id", event.ID), slog.String("event_type", event.EventType))
		return nil
	}
	if err := h.articleSearch.DeleteArticle(ctx, payload.TargetID); err != nil {
		return err
	}
	h.log.InfoContext(ctx, "article_removed_from_search", slog.Int64("outbox_id", event.ID), slog.String("article_id", payload.TargetID))
	return nil
}

func (h *NotificationHandler) indexPublishedArticle(ctx context.Context, articleID string) error {
	var document search.ArticleDocument
	var content []byte
	if err := h.db.QueryRow(ctx, `
		select
			a.id::text,
			a.author_id::text,
			r.title,
			r.excerpt,
			r.content,
			coalesce(array_agg(t.name order by rt.position) filter (where t.id is not null), '{}'),
			a.published_at,
			a.updated_at
		from articles a
		join article_revisions r on r.id = a.published_revision_id
		left join article_revision_tags rt on rt.revision_id = r.id
		left join tags t on t.id = rt.tag_id
		where a.id = $1
			and a.status = 'published'
			and a.published_at <= now()
		group by a.id, r.id
	`, articleID).Scan(
		&document.ID,
		&document.AuthorID,
		&document.Title,
		&document.Excerpt,
		&content,
		&document.Tags,
		&document.PublishedAt,
		&document.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return h.articleSearch.DeleteArticle(ctx, articleID)
		}
		return err
	}
	document.Body = articleBodyText(content)
	if err := h.articleSearch.IndexArticle(ctx, document); err != nil {
		return err
	}
	h.log.InfoContext(ctx, "article_indexed", slog.String("article_id", articleID))
	return nil
}

func articleBodyText(content json.RawMessage) string {
	var decoded any
	if err := json.Unmarshal(content, &decoded); err != nil {
		return string(content)
	}
	parts := make([]string, 0)
	collectArticleText(decoded, &parts)
	return strings.Join(parts, " ")
}

func collectArticleText(value any, parts *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if key == "text" || key == "caption" || key == "description" {
				if text, ok := nested.(string); ok && strings.TrimSpace(text) != "" {
					*parts = append(*parts, strings.TrimSpace(text))
					continue
				}
			}
			collectArticleText(nested, parts)
		}
	case []any:
		for _, nested := range typed {
			collectArticleText(nested, parts)
		}
	}
}
