package events

import (
	"context"
	"encoding/json"

	"catch/apps/api/internal/platform/db"
)

type NotificationInput struct {
	UserID     string
	EventType  string
	TargetType string
	TargetID   string
	Title      string
	Body       string
}

func AddOutbox(ctx context.Context, q db.Querier, aggregateType, aggregateID, eventType string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = q.Exec(ctx, `
		insert into outbox_events (aggregate_type, aggregate_id, event_type, payload)
		values ($1, $2, $3, $4::jsonb)
	`, aggregateType, aggregateID, eventType, string(encoded))
	return err
}

func Notify(ctx context.Context, q db.Querier, input NotificationInput) error {
	var notificationID string
	tag, err := q.Exec(ctx, `
		update notifications
		set
			title = $5,
			body = $6,
			unread_count = unread_count + 1,
			updated_at = now()
		where user_id = $1
			and event_type = $2
			and coalesce(target_type, '') = $3
			and coalesce(target_id, '') = $4
			and read_at is null
	`, input.UserID, input.EventType, input.TargetType, input.TargetID, input.Title, input.Body)
	if err != nil {
		return err
	}
	eventType := "notification.updated"
	if tag.RowsAffected() > 0 {
		if err := q.QueryRow(ctx, `
			select id::text
			from notifications
			where user_id = $1
				and event_type = $2
				and coalesce(target_type, '') = $3
				and coalesce(target_id, '') = $4
				and read_at is null
			order by updated_at desc
			limit 1
		`, input.UserID, input.EventType, input.TargetType, input.TargetID).Scan(&notificationID); err != nil {
			return err
		}
	} else if err := q.QueryRow(ctx, `
		insert into notifications (user_id, event_type, target_type, target_id, title, body)
		values ($1, $2, nullif($3, ''), nullif($4, ''), $5, $6)
		returning id::text
	`, input.UserID, input.EventType, input.TargetType, input.TargetID, input.Title, input.Body).Scan(&notificationID); err != nil {
		return err
	} else {
		eventType = "notification.created"
	}

	return AddOutbox(ctx, q, "notification", notificationID, eventType, map[string]any{
		"notification_id": notificationID,
		"user_id":         input.UserID,
		"event_type":      input.EventType,
		"target_type":     input.TargetType,
		"target_id":       input.TargetID,
	})
}
