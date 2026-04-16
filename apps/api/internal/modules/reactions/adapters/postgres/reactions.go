package postgres

import (
	"context"
	"errors"

	"catch/apps/api/internal/modules/reactions/domain"
	"catch/apps/api/internal/modules/reactions/ports"
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

func (r *Repository) SetReaction(ctx context.Context, input ports.SetReactionInput) (int, error) {
	targetAuthorID, err := r.targetAuthor(ctx, input.TargetType, input.TargetID)
	if err != nil {
		return 0, err
	}

	oldValue, err := r.currentValue(ctx, input)
	if err != nil {
		return 0, err
	}

	if input.Value == 0 {
		if _, err := r.tx.Querier(ctx).Exec(ctx, `
			delete from reactions
			where target_type = $1 and target_id = $2 and user_id = $3
		`, input.TargetType, input.TargetID, input.ActorID); err != nil {
			return 0, err
		}
	} else {
		if _, err := r.tx.Querier(ctx).Exec(ctx, `
			insert into reactions (target_type, target_id, user_id, value)
			values ($1, $2, $3, $4)
			on conflict (target_type, target_id, user_id) do update
			set value = excluded.value, updated_at = now()
		`, input.TargetType, input.TargetID, input.ActorID, input.Value); err != nil {
			return 0, err
		}
	}

	delta := input.Value - oldValue
	if delta != 0 {
		if err := r.applyRatingDelta(ctx, targetAuthorID, input, delta); err != nil {
			return 0, err
		}
	}
	if input.Value != 0 && delta != 0 && targetAuthorID != input.ActorID {
		if err := events.Notify(ctx, r.tx.Querier(ctx), events.NotificationInput{
			UserID:     targetAuthorID,
			EventType:  "rating.changed",
			TargetType: input.TargetType,
			TargetID:   input.TargetID,
			Title:      "Рейтинг изменился",
			Body:       ratingNotificationBody(input.TargetType, input.Value),
		}); err != nil {
			return 0, err
		}
	}

	return input.Value, nil
}

func (r *Repository) Summary(ctx context.Context, targetType, targetID string) (ports.ReactionSummary, error) {
	var summary ports.ReactionSummary
	if err := r.tx.Querier(ctx).QueryRow(ctx, `
		select
			coalesce(count(*) filter (where value = 1), 0)::int,
			coalesce(count(*) filter (where value = -1), 0)::int,
			coalesce(sum(value), 0)::int
		from reactions
		where target_type = $1 and target_id = $2
	`, targetType, targetID).Scan(&summary.ReactionsUp, &summary.ReactionsDown, &summary.ReactionScore); err != nil {
		return ports.ReactionSummary{}, err
	}
	return summary, nil
}

func (r *Repository) targetAuthor(ctx context.Context, targetType, targetID string) (string, error) {
	var authorID string
	var query string
	switch targetType {
	case string(domain.TargetTypeArticle):
		query = `
			select author_id::text
			from articles
			where id = $1 and status = 'published' and published_at <= now()
		`
	case string(domain.TargetTypeComment):
		query = `
			select author_id::text
			from comments
			where id = $1 and status = 'active'
		`
	default:
		return "", domain.ErrInvalidReaction
	}

	if err := r.tx.Querier(ctx).QueryRow(ctx, query, targetID).Scan(&authorID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrInvalidReaction
		}
		return "", err
	}

	return authorID, nil
}

func (r *Repository) currentValue(ctx context.Context, input ports.SetReactionInput) (int, error) {
	var value int
	if err := r.tx.Querier(ctx).QueryRow(ctx, `
		select value
		from reactions
		where target_type = $1 and target_id = $2 and user_id = $3
	`, input.TargetType, input.TargetID, input.ActorID).Scan(&value); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return value, nil
}

func (r *Repository) applyRatingDelta(ctx context.Context, targetAuthorID string, input ports.SetReactionInput, delta int) error {
	reason := ratingReason(input.TargetType, delta)
	if _, err := r.tx.Querier(ctx).Exec(ctx, `
		insert into rating_events (user_id, source_type, source_id, delta, reason)
		values ($1, $2, $3, $4, $5)
	`, targetAuthorID, input.TargetType+"_reaction", input.TargetID, delta, reason); err != nil {
		return err
	}

	_, err := r.tx.Querier(ctx).Exec(ctx, `
		update users
		set rating = least(1000000, rating + $2), updated_at = now()
		where id = $1
	`, targetAuthorID, delta)
	return err
}

func ratingReason(targetType string, delta int) string {
	if targetType == string(domain.TargetTypeArticle) {
		if delta > 0 {
			return "article_like"
		}
		return "article_dislike"
	}
	if delta > 0 {
		return "comment_like"
	}
	return "comment_dislike"
}

func ratingNotificationBody(targetType string, value int) string {
	action := "оценили"
	if value < 0 {
		action = "оценили отрицательно"
	}
	if targetType == string(domain.TargetTypeArticle) {
		return "Вашу статью " + action + "."
	}
	return "Ваш комментарий " + action + "."
}
