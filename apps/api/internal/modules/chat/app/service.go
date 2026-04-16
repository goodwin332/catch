package app

import (
	"context"
	"errors"
	"time"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/chat/app/dto"
	"catch/apps/api/internal/modules/chat/domain"
	"catch/apps/api/internal/modules/chat/ports"
	"catch/apps/api/internal/platform/db"
	httpx "catch/apps/api/internal/platform/http"
)

type Service struct {
	tx   *db.TxManager
	repo ports.Repository
}

func NewService(tx *db.TxManager, repo ports.Repository) *Service {
	return &Service{tx: tx, repo: repo}
}

func (s *Service) StartConversation(ctx context.Context, actor accessdomain.Principal, request dto.StartConversationRequest) (dto.ConversationResponse, error) {
	if !actor.CanChat() {
		return dto.ConversationResponse{}, httpx.Forbidden("Недостаточно прав для чата")
	}
	recipientID, err := domain.NormalizeRecipient(actor.UserID, request.RecipientID)
	if err != nil {
		return dto.ConversationResponse{}, mapChatError(err)
	}
	var conversation domain.Conversation
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		created, err := s.repo.CreateOrGetDirectConversation(ctx, actor.UserID, recipientID)
		if err != nil {
			return err
		}
		conversation = created
		return nil
	})
	if err != nil {
		return dto.ConversationResponse{}, mapChatError(err)
	}
	return mapConversation(conversation), nil
}

func (s *Service) ListConversations(ctx context.Context, actor accessdomain.Principal, limit int) (dto.ConversationListResponse, error) {
	if !actor.CanChat() {
		return dto.ConversationListResponse{}, httpx.Forbidden("Недостаточно прав для чата")
	}
	conversations, err := s.repo.ListConversations(ctx, actor.UserID, normalizeLimit(limit))
	if err != nil {
		return dto.ConversationListResponse{}, err
	}
	items := make([]dto.ConversationResponse, 0, len(conversations))
	for _, conversation := range conversations {
		items = append(items, mapConversation(conversation))
	}
	return dto.ConversationListResponse{Items: items}, nil
}

func (s *Service) ListMessages(ctx context.Context, actor accessdomain.Principal, conversationID, afterID string, limit int) (dto.MessageListResponse, error) {
	if !actor.CanChat() {
		return dto.MessageListResponse{}, httpx.Forbidden("Недостаточно прав для чата")
	}
	var (
		messages []domain.Message
		err      error
	)
	if afterID == "" {
		messages, err = s.repo.ListMessages(ctx, actor.UserID, conversationID, normalizeLimit(limit))
	} else {
		messages, err = s.repo.ListMessagesAfter(ctx, actor.UserID, conversationID, afterID, normalizeLimit(limit))
	}
	if err != nil {
		return dto.MessageListResponse{}, mapChatError(err)
	}
	items := make([]dto.MessageResponse, 0, len(messages))
	for _, message := range messages {
		items = append(items, mapMessage(message))
	}
	return dto.MessageListResponse{Items: items}, nil
}

func (s *Service) ListMessagesAfter(ctx context.Context, actor accessdomain.Principal, conversationID, afterID string, limit int) (dto.MessageListResponse, error) {
	if !actor.CanChat() {
		return dto.MessageListResponse{}, httpx.Forbidden("Недостаточно прав для чата")
	}
	messages, err := s.repo.ListMessagesAfter(ctx, actor.UserID, conversationID, afterID, normalizeLimit(limit))
	if err != nil {
		return dto.MessageListResponse{}, mapChatError(err)
	}
	items := make([]dto.MessageResponse, 0, len(messages))
	for _, message := range messages {
		items = append(items, mapMessage(message))
	}
	return dto.MessageListResponse{Items: items}, nil
}

func (s *Service) SendMessage(ctx context.Context, actor accessdomain.Principal, conversationID string, request dto.SendMessageRequest) (dto.MessageResponse, error) {
	if !actor.CanChat() {
		return dto.MessageResponse{}, httpx.Forbidden("Недостаточно прав для чата")
	}
	body, err := domain.NormalizeMessage(request.Body)
	if err != nil {
		return dto.MessageResponse{}, mapChatError(err)
	}
	var message domain.Message
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		created, err := s.repo.SendMessage(ctx, actor.UserID, conversationID, body)
		if err != nil {
			return err
		}
		message = created
		return nil
	})
	if err != nil {
		return dto.MessageResponse{}, mapChatError(err)
	}
	return mapMessage(message), nil
}

func (s *Service) MarkRead(ctx context.Context, actor accessdomain.Principal, conversationID string) error {
	if !actor.CanChat() {
		return httpx.Forbidden("Недостаточно прав для чата")
	}
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		return s.repo.MarkRead(ctx, actor.UserID, conversationID)
	})
	return mapChatError(err)
}

func mapConversation(conversation domain.Conversation) dto.ConversationResponse {
	return dto.ConversationResponse{
		ID:          conversation.ID,
		MemberIDs:   conversation.MemberIDs,
		UnreadCount: conversation.UnreadCount,
		CreatedAt:   conversation.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   conversation.UpdatedAt.Format(time.RFC3339),
	}
}

func mapMessage(message domain.Message) dto.MessageResponse {
	var readAt *string
	if message.ReadAt != nil {
		formatted := message.ReadAt.Format(time.RFC3339)
		readAt = &formatted
	}
	return dto.MessageResponse{
		ID:             message.ID,
		ConversationID: message.ConversationID,
		SenderID:       message.SenderID,
		Body:           message.Body,
		Status:         message.Status,
		CreatedAt:      message.CreatedAt.Format(time.RFC3339),
		ReadAt:         readAt,
	}
}

func mapChatError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrInvalidConversation):
		return httpx.ValidationError("Диалог указан некорректно", map[string]any{"conversation": "invalid"})
	case errors.Is(err, domain.ErrInvalidMessage):
		return httpx.ValidationError("Сообщение указано некорректно", map[string]any{"message": "invalid"})
	case errors.Is(err, domain.ErrConversationNotFound):
		return httpx.NewError(404, httpx.CodeNotFound, "Диалог не найден")
	default:
		return err
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
