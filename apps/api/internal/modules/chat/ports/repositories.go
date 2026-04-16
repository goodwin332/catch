package ports

import (
	"context"

	"catch/apps/api/internal/modules/chat/domain"
)

type Repository interface {
	CreateOrGetDirectConversation(context.Context, string, string) (domain.Conversation, error)
	ListConversations(context.Context, string, int) ([]domain.Conversation, error)
	ListMessages(context.Context, string, string, int) ([]domain.Message, error)
	ListMessagesAfter(context.Context, string, string, string, int) ([]domain.Message, error)
	SendMessage(context.Context, string, string, string) (domain.Message, error)
	MarkRead(context.Context, string, string) error
}
