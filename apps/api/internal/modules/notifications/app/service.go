package app

import (
	"context"
	"strings"
	"time"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/notifications/app/dto"
	"catch/apps/api/internal/modules/notifications/domain"
	"catch/apps/api/internal/modules/notifications/ports"
	httpx "catch/apps/api/internal/platform/http"
)

type Service struct {
	repo ports.Repository
}

func NewService(repo ports.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, actor accessdomain.Principal, limit int) (dto.NotificationListResponse, error) {
	notifications, err := s.repo.List(ctx, actor.UserID, normalizeLimit(limit))
	if err != nil {
		return dto.NotificationListResponse{}, err
	}
	unreadTotal, err := s.repo.UnreadCount(ctx, actor.UserID)
	if err != nil {
		return dto.NotificationListResponse{}, err
	}
	items := make([]dto.NotificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		items = append(items, mapNotification(notification))
	}
	return dto.NotificationListResponse{Items: items, UnreadTotal: unreadTotal}, nil
}

func (s *Service) UnreadCount(ctx context.Context, actor accessdomain.Principal) (dto.UnreadCountResponse, error) {
	count, err := s.repo.UnreadCount(ctx, actor.UserID)
	if err != nil {
		return dto.UnreadCountResponse{}, err
	}
	return dto.UnreadCountResponse{UnreadTotal: count}, nil
}

func (s *Service) MarkRead(ctx context.Context, actor accessdomain.Principal, notificationID string) error {
	if strings.TrimSpace(notificationID) == "" {
		return httpx.ValidationError("Уведомление указано некорректно", map[string]any{"notification_id": "required"})
	}
	return s.repo.MarkRead(ctx, actor.UserID, notificationID)
}

func (s *Service) MarkTargetRead(ctx context.Context, actor accessdomain.Principal, targetType, targetID string) error {
	cleanTargetType := strings.TrimSpace(targetType)
	cleanTargetID := strings.TrimSpace(targetID)
	if cleanTargetType == "" || cleanTargetID == "" {
		return httpx.ValidationError("Цель уведомления указана некорректно", map[string]any{
			"target_type": "required",
			"target_id":   "required",
		})
	}
	return s.repo.MarkTargetRead(ctx, actor.UserID, cleanTargetType, cleanTargetID)
}

func mapNotification(notification domain.Notification) dto.NotificationResponse {
	var readAt *string
	if notification.ReadAt != nil {
		formatted := notification.ReadAt.Format(time.RFC3339)
		readAt = &formatted
	}
	return dto.NotificationResponse{
		ID:          notification.ID,
		EventType:   notification.EventType,
		TargetType:  notification.TargetType,
		TargetID:    notification.TargetID,
		Title:       notification.Title,
		Body:        notification.Body,
		UnreadCount: notification.UnreadCount,
		ReadAt:      readAt,
		CreatedAt:   notification.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   notification.UpdatedAt.Format(time.RFC3339),
	}
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}
