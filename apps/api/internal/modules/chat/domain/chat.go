package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	MessageStatusSent = "sent"
	MessageStatusRead = "read"
	MaxMessageLength  = 4000
)

var (
	ErrInvalidConversation  = errors.New("invalid conversation")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrInvalidMessage       = errors.New("invalid message")
)

type Conversation struct {
	ID          string
	MemberIDs   []string
	UnreadCount int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Message struct {
	ID             string
	ConversationID string
	SenderID       string
	Body           string
	Status         string
	CreatedAt      time.Time
	ReadAt         *time.Time
}

func NormalizeRecipient(userID, recipientID string) (string, error) {
	clean := strings.TrimSpace(recipientID)
	if clean == "" || clean == userID {
		return "", ErrInvalidConversation
	}
	return clean, nil
}

func NormalizeMessage(body string) (string, error) {
	clean := strings.TrimSpace(body)
	if clean == "" || len([]rune(clean)) > MaxMessageLength {
		return "", ErrInvalidMessage
	}
	return clean, nil
}
