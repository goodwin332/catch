package ports

import (
	"context"

	"catch/apps/api/internal/modules/notifications/domain"
)

type Repository interface {
	List(context.Context, string, int) ([]domain.Notification, error)
	UnreadCount(context.Context, string) (int, error)
	MarkRead(context.Context, string, string) error
	MarkTargetRead(context.Context, string, string, string) error
}
