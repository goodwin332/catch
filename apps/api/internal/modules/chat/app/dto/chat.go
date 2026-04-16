package dto

type StartConversationRequest struct {
	RecipientID string `json:"recipient_id"`
}

type SendMessageRequest struct {
	Body string `json:"body"`
}

type ConversationResponse struct {
	ID          string   `json:"id"`
	MemberIDs   []string `json:"member_ids"`
	UnreadCount int      `json:"unread_count"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type ConversationListResponse struct {
	Items []ConversationResponse `json:"items"`
}

type MessageResponse struct {
	ID             string  `json:"id"`
	ConversationID string  `json:"conversation_id"`
	SenderID       string  `json:"sender_id"`
	Body           string  `json:"body"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"created_at"`
	ReadAt         *string `json:"read_at,omitempty"`
}

type MessageListResponse struct {
	Items []MessageResponse `json:"items"`
}
