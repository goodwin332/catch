package postgres

import (
	"context"
	"errors"

	"catch/apps/api/internal/modules/chat/domain"
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

func (r *Repository) CreateOrGetDirectConversation(ctx context.Context, userID, recipientID string) (domain.Conversation, error) {
	existingID, err := r.findDirectConversation(ctx, userID, recipientID)
	if err == nil {
		return r.loadConversation(ctx, userID, existingID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Conversation{}, err
	}

	var conversationID string
	if err := r.tx.Querier(ctx).QueryRow(ctx, `
		insert into chat_conversations default values
		returning id::text
	`).Scan(&conversationID); err != nil {
		return domain.Conversation{}, err
	}
	if _, err := r.tx.Querier(ctx).Exec(ctx, `
		insert into chat_conversation_members (conversation_id, user_id)
		values ($1, $2), ($1, $3)
	`, conversationID, userID, recipientID); err != nil {
		return domain.Conversation{}, err
	}
	return r.loadConversation(ctx, userID, conversationID)
}

func (r *Repository) ListConversations(ctx context.Context, userID string, limit int) ([]domain.Conversation, error) {
	rows, err := r.tx.Querier(ctx).Query(ctx, `
		with selected as (
			select c.id, c.created_at, c.updated_at,
				(
					select count(*)::int
					from chat_messages m
					where m.conversation_id = c.id
						and m.sender_id <> $1
						and (
							am.last_read_message_id is null
							or m.created_at > (
								select lm.created_at from chat_messages lm where lm.id = am.last_read_message_id
							)
						)
				) as unread_count
			from chat_conversations c
			join chat_conversation_members am on am.conversation_id = c.id and am.user_id = $1
			order by c.updated_at desc
			limit $2
		)
		select selected.id::text, selected.created_at, selected.updated_at, selected.unread_count, cm.user_id::text
		from selected
		join chat_conversation_members cm on cm.conversation_id = selected.id
		order by selected.updated_at desc, cm.created_at asc
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanConversations(rows)
}

func (r *Repository) ListMessages(ctx context.Context, userID, conversationID string, limit int) ([]domain.Message, error) {
	if err := r.ensureMember(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	rows, err := r.tx.Querier(ctx).Query(ctx, `
		select id::text, conversation_id::text, sender_id::text, body, status, created_at, read_at
		from (
			select id, conversation_id, sender_id, body, status, created_at, read_at
			from chat_messages
			where conversation_id = $1
			order by created_at desc, id desc
			limit $2
		) limited
		order by created_at asc, id asc
	`, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]domain.Message, 0)
	for rows.Next() {
		message, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *Repository) ListMessagesAfter(ctx context.Context, userID, conversationID, afterID string, limit int) ([]domain.Message, error) {
	if err := r.ensureMember(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	if afterID == "" {
		return r.ListMessages(ctx, userID, conversationID, limit)
	}

	rows, err := r.tx.Querier(ctx).Query(ctx, `
		with anchor as (
			select created_at
			from chat_messages
			where id = $3 and conversation_id = $1
		)
		select id::text, conversation_id::text, sender_id::text, body, status, created_at, read_at
		from chat_messages
		where conversation_id = $1
			and created_at > (select created_at from anchor)
		order by created_at asc, id asc
		limit $2
	`, conversationID, limit, afterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]domain.Message, 0)
	for rows.Next() {
		message, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *Repository) SendMessage(ctx context.Context, userID, conversationID, body string) (domain.Message, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		insert into chat_messages (conversation_id, sender_id, body)
		select $1, $2, $3
		where exists (
			select 1 from chat_conversation_members
			where conversation_id = $1 and user_id = $2
		)
		returning id::text, conversation_id::text, sender_id::text, body, status, created_at, read_at
	`, conversationID, userID, body)
	message, err := scanMessage(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Message{}, domain.ErrConversationNotFound
		}
		return domain.Message{}, err
	}
	if _, err := r.tx.Querier(ctx).Exec(ctx, `
		update chat_conversations
		set updated_at = now()
		where id = $1
	`, conversationID); err != nil {
		return domain.Message{}, err
	}
	if err := r.notifyMessageCreated(ctx, message); err != nil {
		return domain.Message{}, err
	}
	return message, nil
}

func (r *Repository) MarkRead(ctx context.Context, userID, conversationID string) error {
	if err := r.ensureMember(ctx, userID, conversationID); err != nil {
		return err
	}
	if _, err := r.tx.Querier(ctx).Exec(ctx, `
		update chat_conversation_members
		set last_read_message_id = (
			select id
			from chat_messages
			where conversation_id = $1
			order by created_at desc, id desc
			limit 1
		)
		where conversation_id = $1 and user_id = $2
	`, conversationID, userID); err != nil {
		return err
	}
	_, err := r.tx.Querier(ctx).Exec(ctx, `
		update chat_messages
		set status = 'read', read_at = coalesce(read_at, now())
		where conversation_id = $1
			and sender_id <> $2
			and status = 'sent'
	`, conversationID, userID)
	return err
}

func (r *Repository) findDirectConversation(ctx context.Context, userID, recipientID string) (string, error) {
	var conversationID string
	err := r.tx.Querier(ctx).QueryRow(ctx, `
		select c.id::text
		from chat_conversations c
		join chat_conversation_members self on self.conversation_id = c.id and self.user_id = $1
		join chat_conversation_members recipient on recipient.conversation_id = c.id and recipient.user_id = $2
		where (
			select count(*)
			from chat_conversation_members members
			where members.conversation_id = c.id
		) = 2
		order by c.updated_at desc
		limit 1
	`, userID, recipientID).Scan(&conversationID)
	return conversationID, err
}

func (r *Repository) loadConversation(ctx context.Context, userID, conversationID string) (domain.Conversation, error) {
	rows, err := r.tx.Querier(ctx).Query(ctx, `
		with selected as (
			select c.id, c.created_at, c.updated_at,
				(
					select count(*)::int
					from chat_messages m
					where m.conversation_id = c.id
						and m.sender_id <> $1
						and (
							am.last_read_message_id is null
							or m.created_at > (
								select lm.created_at from chat_messages lm where lm.id = am.last_read_message_id
							)
						)
				) as unread_count
			from chat_conversations c
			join chat_conversation_members am on am.conversation_id = c.id and am.user_id = $1
			where c.id = $2
		)
		select selected.id::text, selected.created_at, selected.updated_at, selected.unread_count, cm.user_id::text
		from selected
		join chat_conversation_members cm on cm.conversation_id = selected.id
		order by cm.created_at asc
	`, userID, conversationID)
	if err != nil {
		return domain.Conversation{}, err
	}
	defer rows.Close()

	conversations, err := scanConversations(rows)
	if err != nil {
		return domain.Conversation{}, err
	}
	if len(conversations) == 0 {
		return domain.Conversation{}, domain.ErrConversationNotFound
	}
	return conversations[0], nil
}

func (r *Repository) ensureMember(ctx context.Context, userID, conversationID string) error {
	var exists bool
	if err := r.tx.Querier(ctx).QueryRow(ctx, `
		select exists (
			select 1 from chat_conversation_members
			where conversation_id = $1 and user_id = $2
		)
	`, conversationID, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return domain.ErrConversationNotFound
	}
	return nil
}

func (r *Repository) notifyMessageCreated(ctx context.Context, message domain.Message) error {
	q := r.tx.Querier(ctx)
	rows, err := q.Query(ctx, `
		select user_id::text
		from chat_conversation_members
		where conversation_id = $1 and user_id <> $2
	`, message.ConversationID, message.SenderID)
	if err != nil {
		return err
	}
	defer rows.Close()

	recipientIDs := make([]string, 0)
	for rows.Next() {
		var recipientID string
		if err := rows.Scan(&recipientID); err != nil {
			return err
		}
		recipientIDs = append(recipientIDs, recipientID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	for _, recipientID := range recipientIDs {
		if err := events.Notify(ctx, q, events.NotificationInput{
			UserID:     recipientID,
			EventType:  "chat.message.created",
			TargetType: "conversation",
			TargetID:   message.ConversationID,
			Title:      "Новое сообщение",
			Body:       "В диалоге появилось новое сообщение.",
		}); err != nil {
			return err
		}
	}
	return events.AddOutbox(ctx, q, "chat_message", message.ID, "chat.message.created", map[string]any{
		"message_id":      message.ID,
		"conversation_id": message.ConversationID,
		"sender_id":       message.SenderID,
	})
}

type rowScanner interface {
	Scan(...any) error
}

func scanMessage(row rowScanner) (domain.Message, error) {
	var message domain.Message
	if err := row.Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&message.Body,
		&message.Status,
		&message.CreatedAt,
		&message.ReadAt,
	); err != nil {
		return domain.Message{}, err
	}
	return message, nil
}

func scanConversations(rows pgx.Rows) ([]domain.Conversation, error) {
	byID := make(map[string]int)
	conversations := make([]domain.Conversation, 0)
	for rows.Next() {
		var conversation domain.Conversation
		var memberID string
		if err := rows.Scan(
			&conversation.ID,
			&conversation.CreatedAt,
			&conversation.UpdatedAt,
			&conversation.UnreadCount,
			&memberID,
		); err != nil {
			return nil, err
		}
		index, ok := byID[conversation.ID]
		if !ok {
			conversation.MemberIDs = []string{memberID}
			conversations = append(conversations, conversation)
			byID[conversation.ID] = len(conversations) - 1
			continue
		}
		conversations[index].MemberIDs = append(conversations[index].MemberIDs, memberID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return conversations, nil
}
