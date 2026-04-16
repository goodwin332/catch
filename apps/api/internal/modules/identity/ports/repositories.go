package ports

import (
	"context"
	"time"

	"catch/apps/api/internal/modules/identity/domain"
)

type UserRepository interface {
	FindByEmail(context.Context, domain.Email) (domain.User, error)
	FindByID(context.Context, string) (domain.User, error)
	FindByOAuthAccount(context.Context, string, string) (domain.User, error)
	CreateEmailUser(context.Context, CreateEmailUserInput) (domain.User, error)
	EnsureDevUser(context.Context, domain.Email) (domain.User, error)
	LinkOAuthAccount(context.Context, LinkOAuthAccountInput) error
}

type SessionRepository interface {
	Create(context.Context, CreateSessionInput) error
	FindUserByTokenHash(context.Context, []byte, time.Time) (domain.SessionUser, error)
	RevokeByTokenHash(context.Context, []byte, time.Time) error
}

type EmailCodeRepository interface {
	Create(context.Context, CreateEmailCodeInput) error
	Consume(context.Context, ConsumeEmailCodeInput) (EmailCodePurpose, error)
	IncrementAttempts(context.Context, domain.Email, time.Time) error
}

type CreateEmailUserInput struct {
	Email       domain.Email
	DisplayName string
}

type LinkOAuthAccountInput struct {
	UserID            string
	Provider          string
	ProviderAccountID string
	Email             domain.Email
}

type CreateSessionInput struct {
	UserID        string
	TokenHash     []byte
	CSRFTokenHash []byte
	UserAgent     string
	IP            string
	ExpiresAt     time.Time
}

type EmailCodePurpose string

const (
	EmailCodePurposeLogin        EmailCodePurpose = "login"
	EmailCodePurposeRegistration EmailCodePurpose = "registration"
)

type CreateEmailCodeInput struct {
	Email     domain.Email
	CodeHash  []byte
	Purpose   EmailCodePurpose
	RequestIP string
	ExpiresAt time.Time
}

type ConsumeEmailCodeInput struct {
	Email    domain.Email
	CodeHash []byte
	Now      time.Time
}
