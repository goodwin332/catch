package domain

import "time"

type User struct {
	ID          string
	Email       Email
	Username    string
	DisplayName string
	AvatarURL   string
	Rating      int
	Role        string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SessionUser struct {
	User          User
	SessionID     string
	CSRFTokenHash []byte
	ExpiresAt     time.Time
}
