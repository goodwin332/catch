package postgres

import (
	"context"
	"errors"
	"strings"

	"catch/apps/api/internal/modules/bookmarks/domain"
	"catch/apps/api/internal/modules/bookmarks/ports"
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

func (r *Repository) ListBookmarkLists(ctx context.Context, userID string) ([]domain.List, error) {
	rows, err := r.tx.Querier(ctx).Query(ctx, `
		select id::text, user_id::text, name, position
		from bookmark_lists
		where user_id = $1
		order by position asc, created_at asc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []domain.List
	for rows.Next() {
		list, err := scanList(rows)
		if err != nil {
			return nil, err
		}
		lists = append(lists, list)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lists, nil
}

func (r *Repository) ListBookmarkedArticles(ctx context.Context, input ports.ListBookmarkedArticlesInput) ([]domain.Article, error) {
	pattern := "%" + strings.ToLower(input.Query) + "%"
	rows, err := r.tx.Querier(ctx).Query(ctx, `
		select bl.id::text, a.id::text, a.author_id::text, r.title, r.excerpt,
			coalesce(array_agg(t.name order by rt.position) filter (where t.id is not null), '{}'),
			a.published_at, bi.created_at
		from bookmark_items bi
		join bookmark_lists bl on bl.id = bi.list_id
		join articles a on a.id = bi.article_id
		join article_revisions r on r.id = a.current_revision_id
		left join article_revision_tags rt on rt.revision_id = r.id
		left join tags t on t.id = rt.tag_id
		where bl.user_id = $1
			and (nullif($2, '') is null or bl.id = nullif($2, '')::uuid)
			and a.status = 'published'
			and a.published_at <= now()
			and (
				$3 = ''
				or lower(r.title) like $4
				or lower(r.excerpt) like $4
				or exists (
					select 1
					from article_revision_tags rt2
					join tags t2 on t2.id = rt2.tag_id
					where rt2.revision_id = r.id and lower(t2.name) like $4
				)
			)
		group by bl.id, a.id, r.id, bi.created_at
		order by bi.created_at desc
		limit $5
	`, input.UserID, input.ListID, input.Query, pattern, input.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles := make([]domain.Article, 0)
	for rows.Next() {
		var article domain.Article
		if err := rows.Scan(
			&article.ListID,
			&article.ArticleID,
			&article.AuthorID,
			&article.Title,
			&article.Excerpt,
			&article.Tags,
			&article.PublishedAt,
			&article.BookmarkedAt,
		); err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return articles, nil
}

func (r *Repository) CreateBookmarkList(ctx context.Context, userID, name string) (domain.List, error) {
	var count int
	if err := r.tx.Querier(ctx).QueryRow(ctx, `select count(*) from bookmark_lists where user_id = $1`, userID).Scan(&count); err != nil {
		return domain.List{}, err
	}
	if count >= 20 {
		return domain.List{}, domain.ErrLimitExceeded
	}

	row := r.tx.Querier(ctx).QueryRow(ctx, `
		insert into bookmark_lists (user_id, name, position)
		values ($1, $2, $3)
		returning id::text, user_id::text, name, position
	`, userID, name, count)
	return scanList(row)
}

func (r *Repository) EnsureDefaultList(ctx context.Context, userID string) (domain.List, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		insert into bookmark_lists (user_id, name, position)
		values ($1, $2, 0)
		on conflict (user_id, (lower(name))) do update set updated_at = bookmark_lists.updated_at
		returning id::text, user_id::text, name, position
	`, userID, domain.DefaultListName)
	return scanList(row)
}

func (r *Repository) AddBookmark(ctx context.Context, userID, listID, articleID string) error {
	var count int
	if err := r.tx.Querier(ctx).QueryRow(ctx, `
		select count(*)
		from bookmark_items bi
		join bookmark_lists bl on bl.id = bi.list_id
		where bl.id = $1 and bl.user_id = $2
	`, listID, userID).Scan(&count); err != nil {
		return err
	}
	if count >= 100 {
		return domain.ErrLimitExceeded
	}

	var authorID string
	if err := r.tx.Querier(ctx).QueryRow(ctx, `
		insert into bookmark_items (list_id, article_id)
		select $1, $3
		where exists (
			select 1 from bookmark_lists where id = $1 and user_id = $2
		)
		and exists (
			select 1 from articles where id = $3 and status = 'published' and published_at <= now()
		)
		on conflict do nothing
		returning (
			select author_id::text from articles where id = $3
		)
	`, listID, userID, articleID).Scan(&authorID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	if authorID == userID {
		return nil
	}
	return events.Notify(ctx, r.tx.Querier(ctx), events.NotificationInput{
		UserID:     authorID,
		EventType:  "bookmark.added",
		TargetType: "article",
		TargetID:   articleID,
		Title:      "Статью добавили в закладки",
		Body:       "Ваш материал сохранили для чтения.",
	})
}

func (r *Repository) RemoveBookmark(ctx context.Context, userID, listID, articleID string) error {
	_, err := r.tx.Querier(ctx).Exec(ctx, `
		delete from bookmark_items bi
		using bookmark_lists bl
		where bi.list_id = bl.id
			and bl.user_id = $1
			and bi.list_id = $2
			and bi.article_id = $3
	`, userID, listID, articleID)
	return err
}

func (r *Repository) Follow(ctx context.Context, followerID, authorID string) (bool, error) {
	tag, err := r.tx.Querier(ctx).Exec(ctx, `
		insert into follows (follower_id, author_id)
		values ($1, $2)
		on conflict do nothing
	`, followerID, authorID)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if err := r.applyFollowRating(ctx, authorID, "follow", 5); err != nil {
		return false, err
	}
	if err := events.Notify(ctx, r.tx.Querier(ctx), events.NotificationInput{
		UserID:     authorID,
		EventType:  "subscription.followed",
		TargetType: "user",
		TargetID:   followerID,
		Title:      "Новый подписчик",
		Body:       "На ваш профиль подписались.",
	}); err != nil {
		return false, err
	}
	return true, nil
}

func (r *Repository) Unfollow(ctx context.Context, followerID, authorID string) (bool, error) {
	tag, err := r.tx.Querier(ctx).Exec(ctx, `
		delete from follows
		where follower_id = $1 and author_id = $2
	`, followerID, authorID)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	return true, r.applyFollowRating(ctx, authorID, "unfollow", -5)
}

func (r *Repository) applyFollowRating(ctx context.Context, authorID, reason string, delta int) error {
	if _, err := r.tx.Querier(ctx).Exec(ctx, `
		insert into rating_events (user_id, source_type, source_id, delta, reason)
		values ($1, 'follow', $2, $3, $4)
	`, authorID, authorID, delta, reason); err != nil {
		return err
	}
	_, err := r.tx.Querier(ctx).Exec(ctx, `
		update users
		set rating = least(1000000, rating + $2), updated_at = now()
		where id = $1
	`, authorID, delta)
	return err
}

type listScanner interface {
	Scan(...any) error
}

func scanList(row listScanner) (domain.List, error) {
	var list domain.List
	if err := row.Scan(&list.ID, &list.UserID, &list.Name, &list.Position); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.List{}, domain.ErrLimitExceeded
		}
		return domain.List{}, err
	}
	return list, nil
}
