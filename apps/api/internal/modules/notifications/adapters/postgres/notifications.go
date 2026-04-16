package postgres

import (
	"context"

	"catch/apps/api/internal/modules/notifications/domain"
	"catch/apps/api/internal/platform/db"
)

type Repository struct {
	tx *db.TxManager
}

func NewRepository(tx *db.TxManager) *Repository {
	return &Repository{tx: tx}
}

func (r *Repository) List(ctx context.Context, userID string, limit int) ([]domain.Notification, error) {
	rows, err := r.tx.Querier(ctx).Query(ctx, `
		select id::text, user_id::text, event_type, coalesce(target_type, ''), coalesce(target_id, ''),
			title, body, unread_count, read_at, created_at, updated_at
		from notifications
		where user_id = $1
		order by updated_at desc
		limit $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifications := make([]domain.Notification, 0)
	for rows.Next() {
		var notification domain.Notification
		if err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.EventType,
			&notification.TargetType,
			&notification.TargetID,
			&notification.Title,
			&notification.Body,
			&notification.UnreadCount,
			&notification.ReadAt,
			&notification.CreatedAt,
			&notification.UpdatedAt,
		); err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return notifications, nil
}

func (r *Repository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	if err := r.tx.Querier(ctx).QueryRow(ctx, `
		select coalesce(sum(unread_count), 0)::int
		from notifications
		where user_id = $1 and read_at is null
	`, userID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) MarkRead(ctx context.Context, userID, notificationID string) error {
	_, err := r.tx.Querier(ctx).Exec(ctx, `
		update notifications
		set read_at = coalesce(read_at, now()), updated_at = now()
		where id = $1 and user_id = $2
	`, notificationID, userID)
	return err
}

func (r *Repository) MarkTargetRead(ctx context.Context, userID, targetType, targetID string) error {
	_, err := r.tx.Querier(ctx).Exec(ctx, `
		update notifications
		set read_at = coalesce(read_at, now()), updated_at = now()
		where user_id = $1
			and target_type = $2
			and target_id = $3
			and read_at is null
	`, userID, targetType, targetID)
	return err
}
