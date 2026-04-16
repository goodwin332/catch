package postgres

import (
	"context"
	"errors"

	"catch/apps/api/internal/modules/comments/domain"
	"catch/apps/api/internal/modules/comments/ports"
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

func (r *Repository) ListByArticle(ctx context.Context, articleID string) ([]domain.Comment, error) {
	rows, err := r.tx.Querier(ctx).Query(ctx, `
		select `+commentSelectColumns("c")+`
		from comments
		c
		where article_id = $1
		order by created_at asc, id asc
	`, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []domain.Comment
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}

func (r *Repository) FindByID(ctx context.Context, commentID string) (domain.Comment, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		select `+commentSelectColumns("c")+`
		from comments c
		where c.id = $1
	`, commentID)
	return scanComment(row)
}

func (r *Repository) Create(ctx context.Context, input ports.CreateCommentInput) (domain.Comment, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		insert into comments (article_id, author_id, parent_id, body)
		select $1, $2, nullif($3, '')::uuid, $4
		where exists (
			select 1 from articles
			where id = $1 and status = 'published' and published_at <= now()
		)
		and (
			nullif($3, '') is null
			or exists (
				select 1 from comments
				where id = nullif($3, '')::uuid and article_id = $1
			)
		)
		returning id::text, article_id::text, author_id::text, coalesce(parent_id::text, ''), body, status, created_at, updated_at, edited_at, 0, 0, 0
	`, input.ArticleID, input.AuthorID, input.ParentID, input.Body)
	comment, err := scanComment(row)
	if err != nil {
		return domain.Comment{}, err
	}
	if err := r.notifyCommentCreated(ctx, comment); err != nil {
		return domain.Comment{}, err
	}
	return comment, nil
}

func (r *Repository) UpdateBody(ctx context.Context, input ports.UpdateCommentInput) (domain.Comment, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		update comments
		set body = $2, edited_at = now(), updated_at = now()
		where id = $1 and status = 'active'
		returning id::text, article_id::text, author_id::text, coalesce(parent_id::text, ''), body, status, created_at, updated_at, edited_at,
			coalesce((select count(*)::int from reactions rr where rr.target_type = 'comment' and rr.target_id = comments.id and rr.value = 1), 0),
			coalesce((select count(*)::int from reactions rr where rr.target_type = 'comment' and rr.target_id = comments.id and rr.value = -1), 0),
			coalesce((select sum(rr.value)::int from reactions rr where rr.target_type = 'comment' and rr.target_id = comments.id), 0)
	`, input.CommentID, input.Body)
	return scanComment(row)
}

func (r *Repository) notifyCommentCreated(ctx context.Context, comment domain.Comment) error {
	q := r.tx.Querier(ctx)
	var articleAuthorID string
	if err := q.QueryRow(ctx, `
		select author_id::text
		from articles
		where id = $1
	`, comment.ArticleID).Scan(&articleAuthorID); err != nil {
		return err
	}
	if articleAuthorID != comment.AuthorID {
		if err := events.Notify(ctx, q, events.NotificationInput{
			UserID:     articleAuthorID,
			EventType:  "comment.created",
			TargetType: "article",
			TargetID:   comment.ArticleID,
			Title:      "Новый комментарий",
			Body:       "К вашей статье добавили комментарий.",
		}); err != nil {
			return err
		}
	}
	return events.AddOutbox(ctx, q, "comment", comment.ID, "comment.created", map[string]any{
		"comment_id": comment.ID,
		"article_id": comment.ArticleID,
		"author_id":  comment.AuthorID,
	})
}

type commentScanner interface {
	Scan(...any) error
}

func scanComment(row commentScanner) (domain.Comment, error) {
	var comment domain.Comment
	if err := row.Scan(
		&comment.ID,
		&comment.ArticleID,
		&comment.AuthorID,
		&comment.ParentID,
		&comment.Body,
		&comment.Status,
		&comment.CreatedAt,
		&comment.UpdatedAt,
		&comment.EditedAt,
		&comment.ReactionsUp,
		&comment.ReactionsDown,
		&comment.ReactionScore,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Comment{}, domain.ErrCommentNotFound
		}
		return domain.Comment{}, err
	}
	return comment, nil
}

func commentSelectColumns(alias string) string {
	return alias + `.id::text,
		` + alias + `.article_id::text,
		` + alias + `.author_id::text,
		coalesce(` + alias + `.parent_id::text, ''),
		` + alias + `.body,
		` + alias + `.status,
		` + alias + `.created_at,
		` + alias + `.updated_at,
		` + alias + `.edited_at,
		coalesce((select count(*)::int from reactions rr where rr.target_type = 'comment' and rr.target_id = ` + alias + `.id and rr.value = 1), 0),
		coalesce((select count(*)::int from reactions rr where rr.target_type = 'comment' and rr.target_id = ` + alias + `.id and rr.value = -1), 0),
		coalesce((select sum(rr.value)::int from reactions rr where rr.target_type = 'comment' and rr.target_id = ` + alias + `.id), 0)`
}
