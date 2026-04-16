package domain

import "time"

type Notification struct {
	ID          string
	UserID      string
	EventType   string
	TargetType  string
	TargetID    string
	Title       string
	Body        string
	UnreadCount int
	ReadAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
