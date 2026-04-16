package dto

type NotificationResponse struct {
	ID          string  `json:"id"`
	EventType   string  `json:"event_type"`
	TargetType  string  `json:"target_type,omitempty"`
	TargetID    string  `json:"target_id,omitempty"`
	Title       string  `json:"title"`
	Body        string  `json:"body"`
	UnreadCount int     `json:"unread_count"`
	ReadAt      *string `json:"read_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type NotificationListResponse struct {
	Items       []NotificationResponse `json:"items"`
	UnreadTotal int                    `json:"unread_total"`
}

type UnreadCountResponse struct {
	UnreadTotal int `json:"unread_total"`
}

type MarkTargetReadRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
}
