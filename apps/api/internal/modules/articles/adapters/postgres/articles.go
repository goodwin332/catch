package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"catch/apps/api/internal/modules/articles/domain"
	"catch/apps/api/internal/modules/articles/ports"
	"catch/apps/api/internal/platform/db"
	"catch/apps/api/internal/platform/events"

	"github.com/jackc/pgx/v5"
)

type Repository struct {
	tx *db.TxManager
}

func NewRepository(tx *db.TxManager) *Repository {
	return &Repository{tx: tx}
}

func (r *Repository) CreateDraft(ctx context.Context, input ports.CreateDraftInput) (domain.Draft, error) {
	q := r.tx.Querier(ctx)

	var articleID string
	if err := q.QueryRow(ctx, `
		insert into articles (author_id, status)
		values ($1, 'draft')
		returning id::text
	`, input.AuthorID).Scan(&articleID); err != nil {
		return domain.Draft{}, err
	}

	revisionID, err := r.createRevision(ctx, articleID, input.AuthorID, 1, input.Title, input.Content, input.Excerpt, domain.RevisionStatusDraft)
	if err != nil {
		return domain.Draft{}, err
	}

	if err := r.syncRevisionTags(ctx, revisionID, input.Tags); err != nil {
		return domain.Draft{}, err
	}
	if err := r.syncRevisionMediaFiles(ctx, revisionID, input.AuthorID, input.Content); err != nil {
		return domain.Draft{}, err
	}

	if _, err := q.Exec(ctx, `
		update articles
		set current_revision_id = $2, updated_at = now()
		where id = $1
	`, articleID, revisionID); err != nil {
		return domain.Draft{}, err
	}

	return r.FindDraftForAuthor(ctx, articleID, input.AuthorID)
}

func (r *Repository) FindDraftForAuthor(ctx context.Context, articleID, authorID string) (domain.Draft, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, draftSelectSQL()+`
		where a.id = $1 and a.author_id = $2 and a.status <> 'removed'
		group by a.id, r.id
	`, articleID, authorID)
	return scanDraft(row)
}

func (r *Repository) ListForAuthor(ctx context.Context, authorID string, limit int) ([]domain.Draft, error) {
	rows, err := r.tx.Querier(ctx).Query(ctx, draftSelectSQL()+`
		where a.author_id = $1 and a.status <> 'removed'
		group by a.id, r.id
		order by a.updated_at desc, a.id desc
		limit $2
	`, authorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDrafts(rows)
}

func (r *Repository) FindPublished(ctx context.Context, articleID string, now time.Time) (domain.Draft, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, draftSelectSQL()+`
		where a.id = $1
			and a.status = 'published'
			and a.published_at <= $2
		group by a.id, r.id
	`, articleID, now)
	return scanDraft(row)
}

func (r *Repository) ListPublished(ctx context.Context, input ports.ListPublishedInput) ([]domain.Draft, error) {
	rows, err := r.tx.Querier(ctx).Query(ctx, draftSelectSQL()+`
		where a.status = 'published'
			and a.published_at <= $1
			and (
				$3::timestamptz is null
				or (a.published_at, a.id) < ($3::timestamptz, $4::uuid)
			)
		group by a.id, r.id
		order by a.published_at desc, a.id desc
		limit $2
	`, input.Now, input.Limit, cursorPublishedAt(input.Cursor), cursorID(input.Cursor))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDrafts(rows)
}

func (r *Repository) ListPublishedByIDs(ctx context.Context, input ports.ListPublishedByIDsInput) ([]domain.Draft, error) {
	if len(input.IDs) == 0 {
		return []domain.Draft{}, nil
	}
	rows, err := r.tx.Querier(ctx).Query(ctx, draftSelectSQL()+`
		where a.id::text = any($1::text[])
			and a.status = 'published'
			and a.published_at <= $2
		group by a.id, r.id
		order by array_position($1::text[], a.id::text)
	`, input.IDs, input.Now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDrafts(rows)
}

func (r *Repository) ListPopular(ctx context.Context, input ports.ListPopularInput) ([]domain.Draft, error) {
	popularitySQL := `(
		coalesce((select sum(rr.value)::int from reactions rr where rr.target_type = 'article' and rr.target_id = a.id), 0) * 5
		+ coalesce((select count(*)::int from comments c where c.article_id = a.id and c.status = 'active'), 0)
	)`
	rows, err := r.tx.Querier(ctx).Query(ctx, draftSelectSQLWithRank(popularitySQL)+`
		where a.status = 'published'
			and a.published_at <= $1
			and a.published_at >= $2
		group by a.id, r.id
		order by
			`+popularitySQL+` desc,
			a.published_at desc,
			a.id desc
		limit $3
	`, input.Now, input.Since, input.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDrafts(rows)
}

func (r *Repository) ListPersonalizedFeed(ctx context.Context, input ports.PersonalizedFeedInput) ([]domain.Draft, error) {
	rankSQL := `case when exists (
			select 1 from follows f
			where f.follower_id = $2 and f.author_id = a.author_id
		) then 0 else 1 end`
	rows, err := r.tx.Querier(ctx).Query(ctx, draftSelectSQLWithRank(rankSQL)+`
		where a.status = 'published'
			and a.published_at <= $1
			and (
				$4::integer is null
				or `+rankSQL+` > $4::integer
				or (`+rankSQL+` = $4::integer and a.published_at < $5::timestamptz)
				or (`+rankSQL+` = $4::integer and a.published_at = $5::timestamptz and a.id < $6::uuid)
			)
		group by a.id, r.id
		order by
			`+rankSQL+` asc,
			a.published_at desc,
			a.id desc
		limit $3
	`, input.Now, input.UserID, input.Limit, cursorRank(input.Cursor), cursorPublishedAt(input.Cursor), cursorID(input.Cursor))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDrafts(rows)
}

func (r *Repository) SearchPublished(ctx context.Context, input ports.SearchPublishedInput) ([]domain.Draft, error) {
	pattern := "%" + strings.ToLower(input.Query) + "%"
	rankSQL := `case when lower(r.title) like $2 then 0 else 1 end`
	rows, err := r.tx.Querier(ctx).Query(ctx, draftSelectSQLWithRank(rankSQL)+`
		where a.status = 'published'
			and a.published_at <= $1
			and (
				lower(r.title) like $2
				or lower(r.content::text) like $2
				or exists (
					select 1
					from article_revision_tags rt2
					join tags t2 on t2.id = rt2.tag_id
					where rt2.revision_id = r.id and lower(t2.name) like $2
				)
			)
			and (
				$4::integer is null
				or `+rankSQL+` > $4::integer
				or (`+rankSQL+` = $4::integer and a.published_at < $5::timestamptz)
				or (`+rankSQL+` = $4::integer and a.published_at = $5::timestamptz and a.id < $6::uuid)
			)
		group by a.id, r.id
		order by
			`+rankSQL+` asc,
			a.published_at desc,
			a.id desc
		limit $3
	`, input.Now, pattern, input.Limit, cursorRank(input.Cursor), cursorPublishedAt(input.Cursor), cursorID(input.Cursor))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDrafts(rows)
}

func (r *Repository) UpdateDraftRevision(ctx context.Context, input ports.UpdateDraftRevisionInput) (domain.Draft, error) {
	q := r.tx.Querier(ctx)

	var nextVersion int
	if err := q.QueryRow(ctx, `
		select coalesce(max(r.version), 0) + 1
		from articles a
		join article_revisions r on r.article_id = a.id
		where a.id = $1
			and a.author_id = $2
			and a.status in ('draft', 'archived')
	`, input.ArticleID, input.AuthorID).Scan(&nextVersion); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Draft{}, domain.ErrArticleNotFound
		}
		return domain.Draft{}, err
	}
	if nextVersion == 1 {
		return domain.Draft{}, domain.ErrArticleNotFound
	}

	revisionID, err := r.createRevision(ctx, input.ArticleID, input.AuthorID, nextVersion, input.Title, input.Content, input.Excerpt, domain.RevisionStatusDraft)
	if err != nil {
		return domain.Draft{}, err
	}

	if err := r.syncRevisionTags(ctx, revisionID, input.Tags); err != nil {
		return domain.Draft{}, err
	}
	if err := r.syncRevisionMediaFiles(ctx, revisionID, input.AuthorID, input.Content); err != nil {
		return domain.Draft{}, err
	}

	if _, err := q.Exec(ctx, `
		update articles
		set
			status = 'draft',
			current_revision_id = $3,
			moderation_required = false,
			scheduled_at = null,
			published_at = null,
			updated_at = now()
		where id = $1 and author_id = $2
	`, input.ArticleID, input.AuthorID, revisionID); err != nil {
		return domain.Draft{}, err
	}

	return r.FindDraftForAuthor(ctx, input.ArticleID, input.AuthorID)
}

func (r *Repository) SubmitDraft(ctx context.Context, input ports.SubmitDraftInput) (domain.Draft, error) {
	q := r.tx.Querier(ctx)

	tag, err := q.Exec(ctx, `
		update article_revisions r
		set status = $3
		from articles a
		where r.id = a.current_revision_id
			and a.id = $1
			and a.author_id = $2
			and a.status in ('draft', 'ready_to_publish')
	`, input.ArticleID, input.AuthorID, string(input.RevisionStatus))
	if err != nil {
		return domain.Draft{}, err
	}
	if tag.RowsAffected() == 0 {
		return domain.Draft{}, domain.ErrArticleNotEditable
	}

	if _, err := q.Exec(ctx, `
		update articles
		set
			status = $3,
			moderation_required = $4,
			scheduled_at = $5,
			published_at = $6,
			published_revision_id = case when $6::timestamptz is null then published_revision_id else current_revision_id end,
			updated_at = now()
		where id = $1 and author_id = $2
	`, input.ArticleID, input.AuthorID, string(input.ArticleStatus), input.ModerationRequired, input.ScheduledAt, input.PublishedAt); err != nil {
		return domain.Draft{}, err
	}

	if input.ArticleStatus == domain.ArticleStatusInModeration {
		if _, err := q.Exec(ctx, `
			insert into moderation_submissions (article_id, revision_id, author_id)
			select id, current_revision_id, author_id
			from articles
			where id = $1 and author_id = $2 and current_revision_id is not null
			on conflict (revision_id) do nothing
		`, input.ArticleID, input.AuthorID); err != nil {
			return domain.Draft{}, err
		}
	}
	if input.ArticleStatus == domain.ArticleStatusPublished {
		if err := r.notifyArticlePublished(ctx, input.ArticleID, input.AuthorID); err != nil {
			return domain.Draft{}, err
		}
	}

	return r.FindDraftForAuthor(ctx, input.ArticleID, input.AuthorID)
}

func (r *Repository) notifyArticlePublished(ctx context.Context, articleID, authorID string) error {
	q := r.tx.Querier(ctx)
	var title string
	if err := q.QueryRow(ctx, `
		select r.title
		from articles a
		join article_revisions r on r.id = a.current_revision_id
		where a.id = $1 and a.author_id = $2
	`, articleID, authorID).Scan(&title); err != nil {
		return err
	}

	if err := events.AddOutbox(ctx, q, "article", articleID, "article.published", map[string]any{
		"article_id": articleID,
		"author_id":  authorID,
		"title":      title,
	}); err != nil {
		return err
	}
	return nil
}

func (r *Repository) createRevision(ctx context.Context, articleID, authorID string, version int, title string, content json.RawMessage, excerpt string, status domain.RevisionStatus) (string, error) {
	var revisionID string
	if err := r.tx.Querier(ctx).QueryRow(ctx, `
		insert into article_revisions (article_id, author_id, version, title, content, excerpt, status)
		values ($1, $2, $3, $4, $5::jsonb, $6, $7)
		returning id::text
	`, articleID, authorID, version, title, string(content), excerpt, string(status)).Scan(&revisionID); err != nil {
		return "", err
	}
	return revisionID, nil
}

func (r *Repository) syncRevisionTags(ctx context.Context, revisionID string, tags []string) error {
	q := r.tx.Querier(ctx)
	if _, err := q.Exec(ctx, `delete from article_revision_tags where revision_id = $1`, revisionID); err != nil {
		return err
	}

	for position, name := range tags {
		slug := tagSlug(name)
		var tagID string
		if err := q.QueryRow(ctx, `
			insert into tags (name, slug)
			values ($1, $2)
			on conflict (slug) do update set name = excluded.name
			returning id::text
		`, name, slug).Scan(&tagID); err != nil {
			return err
		}
		if _, err := q.Exec(ctx, `
			insert into article_revision_tags (revision_id, tag_id, position)
			values ($1, $2, $3)
		`, revisionID, tagID, position); err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) syncRevisionMediaFiles(ctx context.Context, revisionID, authorID string, content json.RawMessage) error {
	fileIDs, err := domain.ExtractMediaFileIDs(content)
	if err != nil {
		return err
	}
	if len(fileIDs) == 0 {
		return nil
	}

	q := r.tx.Querier(ctx)
	inserted := 0
	for position, fileID := range fileIDs {
		tag, err := q.Exec(ctx, `
			insert into article_revision_media_files (revision_id, file_id, position)
			select $1, id, $4
			from media_files
			where id = $2
				and uploader_id = $3
				and status = 'ready'
			on conflict (revision_id, file_id) do update set position = excluded.position
		`, revisionID, fileID, authorID, position)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			inserted++
		}
	}
	if inserted != len(fileIDs) {
		return domain.ErrInvalidDocument
	}
	return nil
}

func draftSelectSQL() string {
	return draftSelectSQLWithRank("0")
}

func draftSelectSQLWithRank(rankSQL string) string {
	return `
		select
			a.id::text,
			a.author_id::text,
			a.status,
			coalesce(a.current_revision_id::text, ''),
			coalesce(a.published_revision_id::text, ''),
			a.moderation_required,
			r.title,
			r.content,
			r.excerpt,
			coalesce(array_agg(t.name order by rt.position) filter (where t.id is not null), '{}'),
			r.version,
			a.scheduled_at,
			a.published_at,
			a.created_at,
			a.updated_at,
			coalesce((select count(*)::int from reactions rr where rr.target_type = 'article' and rr.target_id = a.id and rr.value = 1), 0),
			coalesce((select count(*)::int from reactions rr where rr.target_type = 'article' and rr.target_id = a.id and rr.value = -1), 0),
			coalesce((select sum(rr.value)::int from reactions rr where rr.target_type = 'article' and rr.target_id = a.id), 0),
			` + rankSQL + `
		from articles a
		join article_revisions r on r.id = a.current_revision_id
		left join article_revision_tags rt on rt.revision_id = r.id
		left join tags t on t.id = rt.tag_id
	`
}

type draftScanner interface {
	Scan(...any) error
}

func scanDraft(row draftScanner) (domain.Draft, error) {
	var draft domain.Draft
	var status string
	var content []byte
	if err := row.Scan(
		&draft.ID,
		&draft.AuthorID,
		&status,
		&draft.CurrentRevisionID,
		&draft.PublishedRevisionID,
		&draft.ModerationRequired,
		&draft.Title,
		&content,
		&draft.Excerpt,
		&draft.Tags,
		&draft.Version,
		&draft.ScheduledAt,
		&draft.PublishedAt,
		&draft.CreatedAt,
		&draft.UpdatedAt,
		&draft.ReactionsUp,
		&draft.ReactionsDown,
		&draft.ReactionScore,
		&draft.SortRank,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Draft{}, domain.ErrArticleNotFound
		}
		return domain.Draft{}, err
	}
	draft.Status = domain.ArticleStatus(status)
	draft.Content = json.RawMessage(content)
	return draft, nil
}

func scanDrafts(rows pgx.Rows) ([]domain.Draft, error) {
	var drafts []domain.Draft
	for rows.Next() {
		draft, err := scanDraft(rows)
		if err != nil {
			return nil, err
		}
		drafts = append(drafts, draft)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return drafts, nil
}

func cursorPublishedAt(cursor *ports.ListCursor) *time.Time {
	if cursor == nil {
		return nil
	}
	return &cursor.PublishedAt
}

func cursorID(cursor *ports.ListCursor) *string {
	if cursor == nil {
		return nil
	}
	return &cursor.ID
}

func cursorRank(cursor *ports.ListCursor) *int {
	if cursor == nil {
		return nil
	}
	return &cursor.Rank
}

func tagSlug(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
